using System.Text;
using System.Text.Json;
using Notifications.Api.Models;
using Notifications.Api.Services;
using RabbitMQ.Client;
using RabbitMQ.Client.Events;

namespace Notifications.Api.BackgroundServices;

public class RabbitMqConsumerService : BackgroundService
{
    private readonly IConfiguration _configuration;
    private readonly ISmsSender _smsSender;
    private readonly IEmailSender _emailSender;
    private readonly INotificationStore _store;
    private readonly ILogger<RabbitMqConsumerService> _logger;

    public RabbitMqConsumerService(
        IConfiguration configuration,
        ISmsSender smsSender,
        IEmailSender emailSender,
        INotificationStore store,
        ILogger<RabbitMqConsumerService> logger)
    {
        _configuration = configuration;
        _smsSender = smsSender;
        _emailSender = emailSender;
        _store = store;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        string host = _configuration["RabbitMQ:HostName"] ?? "localhost";
        int port = int.TryParse(_configuration["RabbitMQ:Port"], out var p) ? p : 5672;
        string user = _configuration["RabbitMQ:UserName"] ?? "guest";
        string pass = _configuration["RabbitMQ:Password"] ?? "guest";
        string exchangeName = _configuration["RabbitMQ:ExchangeName"] ?? "users.events";
        string queueName = _configuration["RabbitMQ:QueueName"] ?? "notifications.user-events.queue";

        var factory = new ConnectionFactory
        {
            HostName = host,
            Port = port,
            UserName = user,
            Password = pass,
        };

        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                _logger.LogInformation("Connecting to RabbitMQ at {Host}:{Port}...", host, port);

                await using var connection = await factory.CreateConnectionAsync(stoppingToken);
                await using var channel = await connection.CreateChannelAsync(cancellationToken: stoppingToken);

                // Declare topic exchange matching Users service
                await channel.ExchangeDeclareAsync(
                    exchange: exchangeName,
                    type: ExchangeType.Topic,
                    durable: true,
                    autoDelete: false,
                    cancellationToken: stoppingToken
                );

                // Declare durable queue for notification consumer
                await channel.QueueDeclareAsync(
                    queue: queueName,
                    durable: true,
                    exclusive: false,
                    autoDelete: false,
                    arguments: null,
                    cancellationToken: stoppingToken
                );

                // Bind routing keys
                string[] routingKeys = ["user.registered", "user.password.changed", "user.status.changed", "user.deleted"];
                foreach (var routingKey in routingKeys)
                {
                    await channel.QueueBindAsync(
                        queue: queueName,
                        exchange: exchangeName,
                        routingKey: routingKey,
                        cancellationToken: stoppingToken
                    );
                }

                _logger.LogInformation("Subscribed to queue '{Queue}' bound to '{Exchange}'", queueName, exchangeName);

                var consumer = new AsyncEventingBasicConsumer(channel);
                consumer.ReceivedAsync += async (sender, ea) =>
                {
                    try
                    {
                        var body = ea.Body.ToArray();
                        var json = Encoding.UTF8.GetString(body);
                        await HandleMessageAsync(ea.RoutingKey, json, stoppingToken);
                        await channel.BasicAckAsync(ea.DeliveryTag, multiple: false);
                    }
                    catch (Exception ex)
                    {
                        _logger.LogError(ex, "Failed to process message with routing key '{RoutingKey}'", ea.RoutingKey);
                        await channel.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: false);
                    }
                };

                await channel.BasicConsumeAsync(
                    queue: queueName,
                    autoAck: false,
                    consumer: consumer,
                    cancellationToken: stoppingToken
                );

                // Keep connection active until cancellation requested
                var tcs = new TaskCompletionSource();
                stoppingToken.Register(() => tcs.TrySetResult());
                await tcs.Task;
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
            {
                break;
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "RabbitMQ connection error. Retrying in 5 seconds...");
                try
                {
                    await Task.Delay(TimeSpan.FromSeconds(5), stoppingToken);
                }
                catch (OperationCanceledException)
                {
                    break;
                }
            }
        }

        _logger.LogInformation("RabbitMQ consumer service has stopped.");
    }

    private async Task HandleMessageAsync(string routingKey, string json, CancellationToken ct)
    {
        _logger.LogInformation("📥 [RECEIVED EVENT] Key: {RoutingKey} | Payload: {Payload}", routingKey, json);

        var envelope = JsonSerializer.Deserialize<EventEnvelope>(json);
        if (envelope == null)
        {
            _logger.LogWarning("Unable to deserialize EventEnvelope for key {RoutingKey}", routingKey);
            return;
        }

        switch (routingKey)
        {
            case "user.registered":
                await HandleUserRegisteredAsync(envelope, ct);
                break;

            case "user.password.changed":
                await HandleUserPasswordChangedAsync(envelope, ct);
                break;

            case "user.status.changed":
                await HandleUserStatusChangedAsync(envelope, ct);
                break;

            case "user.deleted":
                await HandleUserDeletedAsync(envelope, ct);
                break;

            default:
                _logger.LogInformation("Unhandled routing key: {RoutingKey}", routingKey);
                break;
        }
    }

    private async Task HandleUserRegisteredAsync(EventEnvelope envelope, CancellationToken ct)
    {
        var data = envelope.Data.Deserialize<UserRegisteredPayload>();
        if (data == null) return;

        string smsBody = $"Hello {data.DisplayName}! Welcome to Qomod. Your account is now active.";
        await _smsSender.SendSmsAsync(data.CellphoneNumber, smsBody, ct);

        await _store.AddAsync(new NotificationRecord
        {
            UserId = data.UserId,
            Recipient = data.CellphoneNumber,
            Channel = NotificationChannel.Sms,
            Message = smsBody,
            EventType = envelope.EventType,
            Status = NotificationStatus.Sent,
        }, ct);

        if (!string.IsNullOrWhiteSpace(data.Email))
        {
            string emailSubject = "Welcome to Qomod!";
            string emailBody = $"<p>Hi {data.DisplayName},</p><p>Thank you for joining Qomod. Your user registration was successful.</p>";
            await _emailSender.SendEmailAsync(data.Email, emailSubject, emailBody, ct);

            await _store.AddAsync(new NotificationRecord
            {
                UserId = data.UserId,
                Recipient = data.Email,
                Channel = NotificationChannel.Email,
                Subject = emailSubject,
                Message = emailBody,
                EventType = envelope.EventType,
                Status = NotificationStatus.Sent,
            }, ct);
        }
    }

    private async Task HandleUserPasswordChangedAsync(EventEnvelope envelope, CancellationToken ct)
    {
        var data = envelope.Data.Deserialize<UserPasswordChangedPayload>();
        if (data == null) return;

        string alert = $"Security Alert: The password for your Qomod account was changed at {data.UpdatedAt:yyyy-MM-dd HH:mm:ss} UTC.";
        await _smsSender.SendSmsAsync(envelope.AggregateId, alert, ct);

        await _store.AddAsync(new NotificationRecord
        {
            UserId = envelope.AggregateId,
            Recipient = envelope.AggregateId,
            Channel = NotificationChannel.Sms,
            Message = alert,
            EventType = envelope.EventType,
            Status = NotificationStatus.Sent,
        }, ct);
    }

    private async Task HandleUserStatusChangedAsync(EventEnvelope envelope, CancellationToken ct)
    {
        var data = envelope.Data.Deserialize<UserStatusChangedPayload>();
        if (data == null) return;

        string message = $"Account Notice: Your account status has changed from '{data.OldStatus}' to '{data.NewStatus}'.";
        await _smsSender.SendSmsAsync(envelope.AggregateId, message, ct);

        await _store.AddAsync(new NotificationRecord
        {
            UserId = envelope.AggregateId,
            Recipient = envelope.AggregateId,
            Channel = NotificationChannel.Sms,
            Message = message,
            EventType = envelope.EventType,
            Status = NotificationStatus.Sent,
        }, ct);
    }

    private async Task HandleUserDeletedAsync(EventEnvelope envelope, CancellationToken ct)
    {
        var data = envelope.Data.Deserialize<UserDeletedPayload>();
        if (data == null) return;

        string message = $"Account Notice: Your account has been scheduled for deletion at {data.DeletedAt:yyyy-MM-dd HH:mm:ss} UTC.";
        await _smsSender.SendSmsAsync(envelope.AggregateId, message, ct);

        await _store.AddAsync(new NotificationRecord
        {
            UserId = envelope.AggregateId,
            Recipient = envelope.AggregateId,
            Channel = NotificationChannel.Sms,
            Message = message,
            EventType = envelope.EventType,
            Status = NotificationStatus.Sent,
        }, ct);
    }
}
