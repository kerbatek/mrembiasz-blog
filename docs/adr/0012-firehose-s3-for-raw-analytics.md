# ADR 0012: Use Firehose and S3 for raw analytics events

Date: 2026-08-17

## Status

Accepted

## Context

ADR 0007 chooses SNS as the fanout point for validated analytics events. The
aggregate path already consumes those events through SQS and updates DynamoDB,
but the detailed event history path still needs durable storage for later
analysis.

The backend flow for raw analytics is:

```text
validator Lambda
  -> SNS
  -> Firehose
  -> S3 raw analytics bucket
  -> Athena queries
```

The current validated analytics event is intentionally small:

```json
{"post_slug": "example-post"}
```

## Decision

Subscribe Kinesis Data Firehose directly to the analytics SNS topic.

Firehose will write accepted analytics events to a dedicated S3 bucket. It will
use AWS Glue table metadata for the event schema and convert incoming JSON
records to Parquet before delivery.

The first Glue table has one field:

```text
post_slug string
```

S3 objects will be partitioned by delivery date under `raw/year=.../month=.../day=.../`.

## Rationale

Firehose is the smallest managed hop from SNS to S3 for this workload. It owns
buffering, retries, S3 delivery, and format conversion without adding another
Lambda that only copies records.

Parquet is a better storage format than raw JSON for Athena because it is
columnar and compressed. Keeping the Glue schema explicit also makes schema
changes visible in Terraform review.

The raw analytics bucket is separate from the site and CloudFront log buckets so
retention, query access, and lifecycle rules can change independently later.

## Consequences

Raw analytics delivery is asynchronous and at least once. Duplicate records are
acceptable for first-pass analytics and should be handled by downstream Athena
queries if exact counts matter.

New event fields require updating the Glue table schema before Firehose can
convert them into Parquet columns.

Firehose delivery failures will write to the configured S3 error prefix when
possible. Operational alerting is not part of this first pass.
