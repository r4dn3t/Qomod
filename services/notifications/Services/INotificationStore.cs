using System.Collections.Concurrent;
using Notifications.Api.Models;

namespace Notifications.Api.Services;

public interface INotificationStore
{
    Task AddAsync(NotificationRecord record, CancellationToken cancellationToken = default);
    Task<IReadOnlyList<NotificationRecord>> GetAllAsync(int limit = 50, CancellationToken cancellationToken = default);
    Task<IReadOnlyList<NotificationRecord>> GetByUserIdAsync(string userId, CancellationToken cancellationToken = default);
}

public class InMemoryNotificationStore : INotificationStore
{
    private readonly ConcurrentBag<NotificationRecord> _records = new();

    public Task AddAsync(NotificationRecord record, CancellationToken cancellationToken = default)
    {
        _records.Add(record);
        return Task.CompletedTask;
    }

    public Task<IReadOnlyList<NotificationRecord>> GetAllAsync(int limit = 50, CancellationToken cancellationToken = default)
    {
        var list = _records
            .OrderByDescending(r => r.SentAt)
            .Take(limit)
            .ToList();
        return Task.FromResult<IReadOnlyList<NotificationRecord>>(list);
    }

    public Task<IReadOnlyList<NotificationRecord>> GetByUserIdAsync(string userId, CancellationToken cancellationToken = default)
    {
        var list = _records
            .Where(r => r.UserId.Equals(userId, StringComparison.OrdinalIgnoreCase))
            .OrderByDescending(r => r.SentAt)
            .ToList();
        return Task.FromResult<IReadOnlyList<NotificationRecord>>(list);
    }
}
