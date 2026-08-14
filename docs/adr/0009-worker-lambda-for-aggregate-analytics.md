# ADR 0009: Use Lambda for aggregate analytics processing

Date: 2026-08-14

## Status

Accepted

## Context

ADR 0008 chooses SQS as the buffer between SNS and the aggregate analytics
processor. The next component needs to consume validated post view events from
SQS and update aggregate counters in DynamoDB.

The target backend flow for this branch is:

```text
CloudFront
  -> API Gateway HTTP API
  -> validator Lambda
  -> SNS
  -> SQS
  -> worker Lambda
  -> DynamoDB aggregate post views
```

The processor should stay serverless and should not introduce an always-on
runtime for a workload that is small and bursty.

## Decision

Use an AWS Lambda function as the aggregate analytics worker.

The worker Lambda will be triggered by the aggregate post views SQS queue. It
will parse each message, extract the validated analytics event, and update the
DynamoDB aggregate post view counter.

The Lambda event source mapping will use batch processing with partial batch
failure reporting. Records that update successfully will be acknowledged, while
failed records will remain available for retry and eventually move to the
queue's dead-letter queue if they keep failing.

The worker will update counters with atomic DynamoDB writes instead of reading,
incrementing in memory, and writing the result back.

## Rationale

Lambda matches the workload because aggregate analytics processing is
event-driven, small, and does not need a resident service. It also fits the
existing decision to keep production runtime on managed or serverless AWS
services.

Using SQS as the Lambda event source keeps retry behavior outside the
application code. The worker only needs to report which records failed; SQS and
Lambda handle retry scheduling.

Partial batch failure avoids reprocessing a whole batch when only one message
fails. That keeps retries smaller and reduces duplicate counter updates.

Atomic DynamoDB counter updates avoid lost increments when multiple Lambda
invocations process post view events at the same time.

## Consequences

The worker must treat SQS delivery as at least once. Duplicate messages are
possible, so aggregate counts may overcount if the same post view event is
retried after a successful DynamoDB update but before acknowledgement.

This is acceptable for first-pass blog analytics because the counter is an
operational aggregate, not billing or security data. If exact-once counting
becomes required, the worker will need event IDs and a DynamoDB deduplication
record or equivalent idempotency mechanism.

Worker Lambda errors will show up as SQS retry growth and dead-letter queue
messages. Those queues are the main operational boundary for this processor.

The worker Lambda, event source mapping, and IAM permissions are app-specific
AWS resources and will be managed by Terraform in this repository.
