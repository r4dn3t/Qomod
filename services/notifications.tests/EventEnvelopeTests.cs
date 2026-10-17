using System.Text.Json;
using Notifications.Api.Models;
using Xunit;

namespace Notifications.Tests;

public class EventEnvelopeTests
{
    [Fact]
    public void EventEnvelope_DeserializesGoGeneratedCloudEventCorrectly()
    {
        // Sample JSON emitted from the Users (Go) microservice
        string goJson = """
        {
            "event_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
            "event_type": "user.registered",
            "aggregate_type": "user",
            "aggregate_id": "c1f2e3d4-5678-90ab-cdef-1234567890ab",
            "timestamp": "2026-10-15T00:55:00Z",
            "version": "1.0",
            "data": {
                "user_id": "c1f2e3d4-5678-90ab-cdef-1234567890ab",
                "cellphone_number": "+12025550199",
                "first_name": "Jane",
                "last_name": "Doe",
                "email": "jane.doe@example.com",
                "status": "enabled",
                "created_at": "2026-10-15T00:55:00Z"
            }
        }
        """;

        var envelope = JsonSerializer.Deserialize<EventEnvelope>(goJson);

        Assert.NotNull(envelope);
        Assert.Equal("9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d", envelope.EventId);
        Assert.Equal("user.registered", envelope.EventType);
        Assert.Equal("c1f2e3d4-5678-90ab-cdef-1234567890ab", envelope.AggregateId);

        var data = envelope.Data.Deserialize<UserRegisteredPayload>();
        Assert.NotNull(data);
        Assert.Equal("+12025550199", data.CellphoneNumber);
        Assert.Equal("Jane", data.FirstName);
        Assert.Equal("Jane Doe", data.DisplayName);
    }
}
