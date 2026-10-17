using Notifications.Api.Models;
using Notifications.Api.Services;
using Xunit;

namespace Notifications.Tests;

public class NotificationStoreTests
{
    [Fact]
    public async Task AddAsync_StoresAndRetrievesNotifications()
    {
        // Arrange
        var store = new InMemoryNotificationStore();
        var record1 = new NotificationRecord
        {
            UserId = "user-1",
            Recipient = "+12025550199",
            Channel = NotificationChannel.Sms,
            Message = "Welcome!",
            EventType = "user.registered",
            Status = NotificationStatus.Sent
        };
        var record2 = new NotificationRecord
        {
            UserId = "user-2",
            Recipient = "user2@example.com",
            Channel = NotificationChannel.Email,
            Subject = "Welcome",
            Message = "Hello",
            EventType = "user.registered",
            Status = NotificationStatus.Sent
        };

        // Act
        await store.AddAsync(record1);
        await store.AddAsync(record2);

        var all = await store.GetAllAsync();
        var user1Only = await store.GetByUserIdAsync("user-1");

        // Assert
        Assert.Equal(2, all.Count);
        Assert.Single(user1Only);
        Assert.Equal("user-1", user1Only[0].UserId);
        Assert.Equal(NotificationChannel.Sms, user1Only[0].Channel);
    }

    [Fact]
    public async Task GetByUserIdAsync_CaseInsensitiveMatching()
    {
        // Arrange
        var store = new InMemoryNotificationStore();
        var record = new NotificationRecord
        {
            UserId = "USER-ABC",
            Recipient = "+12025550100",
            Channel = NotificationChannel.Sms,
            Message = "Test",
            EventType = "test",
            Status = NotificationStatus.Sent
        };

        // Act
        await store.AddAsync(record);
        var found = await store.GetByUserIdAsync("user-abc");

        // Assert
        Assert.Single(found);
    }
}
