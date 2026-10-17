namespace Notifications.Api.Services;

public interface IEmailSender
{
    Task<bool> SendEmailAsync(string recipientEmail, string subject, string body, CancellationToken cancellationToken = default);
}

public class LoggingEmailSender : IEmailSender
{
    private readonly ILogger<LoggingEmailSender> _logger;

    public LoggingEmailSender(ILogger<LoggingEmailSender> logger)
    {
        _logger = logger;
    }

    public Task<bool> SendEmailAsync(string recipientEmail, string subject, string body, CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("✉️ [EMAIL DISPATCHED] To: {Recipient} | Subject: {Subject} | Body: {Body}", 
            recipientEmail, subject, body);
        return Task.FromResult(true);
    }
}
