using Notifications.Api.BackgroundServices;
using Notifications.Api.Models;
using Notifications.Api.Services;

var builder = WebApplication.CreateBuilder(args);

// Register Core Services
builder.Services.AddSingleton<ISmsSender, LoggingSmsSender>();
builder.Services.AddSingleton<IEmailSender, LoggingEmailSender>();
builder.Services.AddSingleton<INotificationStore, InMemoryNotificationStore>();

// Register RabbitMQ Consumer as a long-running Hosted Background Service
builder.Services.AddHostedService<RabbitMqConsumerService>();

var app = builder.Build();

// Liveness check
app.MapGet("/healthz", () => Results.Ok(new { status = "ok", service = "notifications" }));

// List dispatched notifications
app.MapGet("/api/notifications", async (INotificationStore store, int? limit, CancellationToken ct) =>
{
    var records = await store.GetAllAsync(limit ?? 50, ct);
    return Results.Ok(new
    {
        success = true,
        count = records.Count,
        data = records
    });
});

// Get notifications for specific user
app.MapGet("/api/notifications/user/{userId}", async (string userId, INotificationStore store, CancellationToken ct) =>
{
    var records = await store.GetByUserIdAsync(userId, ct);
    return Results.Ok(new
    {
        success = true,
        user_id = userId,
        count = records.Count,
        data = records
    });
});

// Manual test trigger endpoint
app.MapPost("/api/notifications/test", async (TestNotificationRequest req, ISmsSender smsSender, INotificationStore store, CancellationToken ct) =>
{
    if (string.IsNullOrWhiteSpace(req.Phone) || string.IsNullOrWhiteSpace(req.Message))
    {
        return Results.BadRequest(new { success = false, error = "phone and message are required" });
    }

    await smsSender.SendSmsAsync(req.Phone, req.Message, ct);
    var record = new NotificationRecord
    {
        UserId = req.UserId ?? "test-user",
        Recipient = req.Phone,
        Channel = NotificationChannel.Sms,
        Message = req.Message,
        EventType = "manual.test",
        Status = NotificationStatus.Sent,
    };
    await store.AddAsync(record, ct);

    return Results.Ok(new { success = true, notification = record });
});

app.Run();

public record TestNotificationRequest(string Phone, string Message, string? UserId);
