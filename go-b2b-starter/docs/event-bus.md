# Event Bus Guide

The event bus enables event-driven architecture for loose coupling between modules using an in-memory publish-subscribe pattern.

## Architecture

**In-memory event bus** - Simple, fast, synchronous
**Publisher-subscriber pattern** - Decouple event producers from consumers
**Type-safe events** - Events are Go structs implementing Event interface

## Core Concepts

### Events

Events represent things that have happened in the system.

**Naming**: Past tense (ResourceCreated, ResourceUpdated, ResourceDeleted)

```go
type ResourceCreatedEvent struct {
    BaseEvent
    ResourceID int32           `json:"resource_id"`
    Name       string          `json:"name"`
    CreatedBy  int32           `json:"created_by"`
    CreatedAt  time.Time       `json:"created_at"`
}
```

### Event Interface

All events implement the Event interface:

```go
type Event interface {
    EventName() string
    EventID() string
    OccurredAt() time.Time
}
```

### BaseEvent

Provides common event fields:

```go
type BaseEvent struct {
    ID         string    `json:"id"`
    Name       string    `json:"name"`
    Timestamp  time.Time `json:"timestamp"`
}
```

## Publishing Events

Emit events when something happens:

```go
func (s *service) CreateResource(ctx context.Context, req *Request) (*Resource, error) {
    // Create resource
    resource, err := s.repo.Create(ctx, req)
    if err != nil {
        return nil, err
    }

    // Publish event
    event := &ResourceCreatedEvent{
        ResourceID: resource.ID,
        Name:       resource.Name,
        CreatedBy:  req.UserID,
        CreatedAt:  resource.CreatedAt,
    }
    s.eventBus.Publish(ctx, event)

    return resource, nil
}
```

**Note**: Publish is fire-and-forget. Failures don't block the operation.

## Subscribing to Events

Listen for events and react:

```go
func (l *ResourceListener) Init(eventBus eventbus.EventBus) {
    // Subscribe to events
    eventBus.Subscribe("resource.created", l.HandleResourceCreated)
    eventBus.Subscribe("resource.updated", l.HandleResourceUpdated)
}

func (l *ResourceListener) HandleResourceCreated(ctx context.Context, event eventbus.Event) error {
    resourceEvent := event.(*ResourceCreatedEvent)

    // React to event
    log.Info("Resource created", zap.Int32("id", resourceEvent.ResourceID))

    // Trigger other actions
    return l.notificationService.NotifyResourceCreated(ctx, resourceEvent.ResourceID)
}
```

## Event Flow

```
Service → Publish Event → Event Bus → Notify Subscribers → Execute Handlers
```

**Synchronous**: Subscribers execute in the same request context
**Ordered**: Subscribers execute in registration order
**Error handling**: Subscriber errors are logged but don't fail the operation

## Common Patterns

### Cross-Module Communication

Module A publishes events, Module B subscribes:

```go
// Module A (Resources)
func (s *resourceService) Delete(ctx context.Context, id int32) error {
    err := s.repo.Delete(ctx, id)
    if err != nil {
        return err
    }

    s.eventBus.Publish(ctx, &ResourceDeletedEvent{ResourceID: id})
    return nil
}

// Module B (Analytics)
func (l *analyticsListener) HandleResourceDeleted(ctx context.Context, event eventbus.Event) error {
    evt := event.(*ResourceDeletedEvent)
    return l.analyticsService.RecordDeletion(ctx, evt.ResourceID)
}
```

### Audit Logging

Subscribe to all events for audit trail:

```go
func (l *auditListener) Init(eventBus eventbus.EventBus) {
    eventBus.Subscribe("*.created", l.HandleCreated)
    eventBus.Subscribe("*.updated", l.HandleUpdated)
    eventBus.Subscribe("*.deleted", l.HandleDeleted)
}

func (l *auditListener) HandleCreated(ctx context.Context, event eventbus.Event) error {
    return l.auditService.Log(ctx, "created", event)
}
```

### Async Processing

Trigger background jobs from events:

```go
func (l *processingListener) HandleFileUploaded(ctx context.Context, event eventbus.Event) error {
    evt := event.(*FileUploadedEvent)

    // Queue async job
    return l.jobQueue.Enqueue(ctx, &ProcessFileJob{
        FileID: evt.FileID,
    })
}
```

## Registration

Register listeners during module initialization:

```go
// internal/resources/cmd/init.go
func Init(container *dig.Container) error {
    return container.Invoke(func(
        eventBus eventbus.EventBus,
        listener *listeners.ResourceListener,
    ) {
        listener.Init(eventBus)
    })
}
```

## File Locations

| Component | Path |
|-----------|------|
| Event bus interface | `internal/eventbus/eventbus.go` |
| Event interface | `internal/eventbus/event.go` |
| Base event | `internal/eventbus/base_event.go` |
| Implementation | `internal/eventbus/memory_eventbus.go` |
| Domain events | `internal/*/domain/events/` |
| Event listeners | `internal/*/domain/listeners/` |

## Next Steps

- **Define events**: Create event structs in `domain/events/`
- **Implement listeners**: Handle events in `domain/listeners/`
- **Publish events**: Emit events in service layer
