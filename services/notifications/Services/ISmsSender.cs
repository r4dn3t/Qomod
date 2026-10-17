namespace Notifications.Api.Services;

public interface ISmsSender
{
    Task<bool> SendSmsAsync(string recipientPhone, string message, CancellationToken cancellationToken = default);
}

public class LoggingSmsSender : ISmsSender
{
    private readonly ILogger<LoggingSmsSender> _logger;

    public LoggingSmsSender(ILogger<LoggingSmsSender> logger)
    {
        _logger = logger;
    }

    public Task<bool> SendSmsAsync(string recipientPhone, string message, CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("📱 [SMS DISPATCHED] To: {Recipient} | Content: {Message}", recipientPhone, message);
        return Task.FromResult(true);
    }
}
