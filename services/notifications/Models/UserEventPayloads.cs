using System.Text.Json.Serialization;

namespace Notifications.Api.Models;

public record UserRegisteredPayload
{
    [JsonPropertyName("user_id")]
    public string UserId { get; init; } = string.Empty;

    [JsonPropertyName("cellphone_number")]
    public string CellphoneNumber { get; init; } = string.Empty;

    [JsonPropertyName("first_name")]
    public string? FirstName { get; init; }

    [JsonPropertyName("last_name")]
    public string? LastName { get; init; }

    [JsonPropertyName("birthdate")]
    public string? Birthdate { get; init; }

    [JsonPropertyName("email")]
    public string? Email { get; init; }

    [JsonPropertyName("status")]
    public string Status { get; init; } = "enabled";

    [JsonPropertyName("created_at")]
    public DateTime CreatedAt { get; init; }

    public string DisplayName =>
        !string.IsNullOrWhiteSpace(FirstName) || !string.IsNullOrWhiteSpace(LastName)
            ? $"{FirstName} {LastName}".Trim()
            : CellphoneNumber;
}

public record UserPasswordChangedPayload
{
    [JsonPropertyName("user_id")]
    public string UserId { get; init; } = string.Empty;

    [JsonPropertyName("updated_at")]
    public DateTime UpdatedAt { get; init; }
}

public record UserStatusChangedPayload
{
    [JsonPropertyName("user_id")]
    public string UserId { get; init; } = string.Empty;

    [JsonPropertyName("old_status")]
    public string OldStatus { get; init; } = string.Empty;

    [JsonPropertyName("new_status")]
    public string NewStatus { get; init; } = string.Empty;

    [JsonPropertyName("updated_at")]
    public DateTime UpdatedAt { get; init; }
}

public record UserDeletedPayload
{
    [JsonPropertyName("user_id")]
    public string UserId { get; init; } = string.Empty;

    [JsonPropertyName("deleted_at")]
    public DateTime DeletedAt { get; init; }
}
