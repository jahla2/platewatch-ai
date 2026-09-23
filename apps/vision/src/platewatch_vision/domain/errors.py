class EventDeliveryError(RuntimeError):
    """Raised when a confirmed detection cannot be delivered after retries."""
