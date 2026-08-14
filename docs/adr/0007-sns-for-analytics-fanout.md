# ADR 0007: Use SNS for analytics fanout

Date: 2026-08-14

## Status

Accepted

## Context

ADR 0006 chooses API Gateway HTTP API as the blog analytics ingestion endpoint.
HTTP API invokes a validator Lambda, and accepted analytics events need to feed
more than one downstream workload.

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

The analytics backend has two separate consumers after validation:

- Aggregate post view counters in DynamoDB.
- Detailed event storage in S3 for Athena queries.

## Decision

Use an SNS standard topic as the fanout point for validated analytics events.

The validator Lambda will publish accepted events to the SNS topic. Downstream
analytics consumers will subscribe to the topic. The aggregate post view
consumer will use SQS for buffering before its worker Lambda. Firehose will
subscribe directly for detailed event delivery to S3.

The first subscribers will support:

1. DynamoDB aggregate post view updates.
2. Detailed analytics event delivery to S3 through Firehose.

## Rationale

SNS matches the fanout problem directly. The validator Lambda should validate
and accept or reject events; it should not know how every analytics consumer
stores or processes those events.

Using SNS keeps adding another analytics consumer from changing the ingestion
path. A new consumer can subscribe to the same validated event stream instead
of adding more work to the validator Lambda.

SNS subscriptions keep the downstream workloads isolated. A temporary failure
in aggregate updates should not block detailed event delivery, and a Firehose
delivery issue should not block DynamoDB counter updates.

A direct Lambda-to-SQS write would work for one consumer, but it would make the
validator Lambda responsible for routing every downstream workload. EventBridge
would add routing features this analytics path does not currently need.

## Consequences

Analytics events become eventually consistent after validation. HTTP API can
return after the validator Lambda publishes to SNS, while downstream consumers
finish later.

Downstream consumers must handle at-least-once delivery because SNS, SQS,
Lambda, and Firehose can retry messages.

The SNS topic and subscriptions are app-specific AWS resources and will be
managed by Terraform in this repository.
