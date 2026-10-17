using System.Text.Json;
using System.Text.Json.Serialization;

namespace Notifications.Api.Models;

/// <summary>
/// CloudEvents-inspired envelope matching the Go users service contract.
/// </summary>
public record EventEnvelope
{
    [JsonPropertyName("event_id")]
    public string EventId { get; init; } = string.Empty;

    [JsonPropertyName("event_type")]
    public string EventType { get; init; } = string.Empty;

    [JsonPropertyName("aggregate_type")]
    public string AggregateType { get; init; } = string.Empty;

    [JsonPropertyName("aggregate_id")]
    public string AggregateId { get; init; } = string.Empty;

    [JsonPropertyName("timestamp")]
    public DateTime Timestamp { get; init; }

    [JsonPropertyName("version")]
    public string Version { get; init; } = "1.0";

    [JsonPropertyName("data")]
    public JsonElement Data { get; init; }
}
