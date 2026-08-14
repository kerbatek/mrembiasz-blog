# ADR 0008: Use SQS for aggregate analytics buffering

Date: 2026-08-14

## Status

Accepted

## Context

ADR 0007 chooses SNS as the fanout point for validated analytics events.
Downstream analytics consumers should process those events independently so a
failure in one workload does not block the others.

The target backend flow is:

```text
CloudFront
  -> API Gateway HTTP API
  -> validator Lambda
  -> SNS
  -> SQS -> worker Lambda -> DynamoDB aggregate post views
  -> Firehose
  -> S3 detailed analytics
  -> Athena queries
```

The aggregate post view workload updates DynamoDB counters. It should tolerate
temporary DynamoDB throttling or worker Lambda failure without forcing retries
back through the ingestion path.

The detailed analytics workload does not need an SQS buffer because Firehose can
subscribe to SNS directly and deliver events to S3.

## Decision

Use an SQS standard queue between SNS and the aggregate analytics worker Lambda.

The aggregate post views queue will subscribe to the SNS topic. The DynamoDB
worker Lambda will consume from that queue. The queue will have a dead-letter
queue for messages that keep failing after retries.

## Rationale

SQS gives the DynamoDB aggregation worker buffering, retries, and failure
isolation. If DynamoDB updates are throttled, detailed event delivery through
Firehose can continue.

Standard queues are enough for analytics events. The system needs durable,
at-least-once delivery, not strict ordering. FIFO queues would add ordering and
deduplication constraints that are not currently needed for post view analytics.

Dead-letter queues keep repeatedly failing messages visible for inspection
without blocking the healthy part of a queue.

Direct SNS-to-Lambda subscriptions would reduce one managed service, but they
would also remove the explicit buffer between fanout and processing. For this
aggregation path, the queue is the simpler failure boundary to operate.

Adding SQS before Firehose is unnecessary because Firehose can subscribe to SNS
directly and already owns buffering for delivery to S3.

## Consequences

Analytics processing is asynchronous and at least once. Worker Lambdas must be
idempotent because the same event can be delivered more than once.

Queue depth and dead-letter queue depth become the main operational signals for
aggregate analytics health.

The SQS queue, dead-letter queue, and SNS subscription are app-specific AWS
resources and will be managed by Terraform in this repository.
