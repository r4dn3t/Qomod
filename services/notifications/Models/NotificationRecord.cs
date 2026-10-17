using System.Text.Json.Serialization;

namespace Notifications.Api.Models;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum NotificationChannel
{
    Sms,
    Email
}

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum NotificationStatus
{
    Sent,
    Failed,
    Simulated
}

public record NotificationRecord
{
    public Guid Id { get; init; } = Guid.NewGuid();
    public string UserId { get; init; } = string.Empty;
    public string Recipient { get; init; } = string.Empty;
    public NotificationChannel Channel { get; init; }
    public string? Subject { get; init; }
    public string Message { get; init; } = string.Empty;
    public string EventType { get; init; } = string.Empty;
    public NotificationStatus Status { get; init; } = NotificationStatus.Sent;
    public DateTime SentAt { get; init; } = DateTime.UtcNow;
}
