# ADR 0010: Use DynamoDB for aggregate post views

Date: 2026-08-14

## Status

Accepted

## Context

ADR 0009 chooses a worker Lambda to consume post view events from SQS and update
aggregate analytics data. The application needs a fast, simple store for
per-post view counts.

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

Detailed analytics events will be stored separately in S3 and queried with
Athena later. The DynamoDB path is only for app-facing aggregates.

## Decision

Use a DynamoDB table for aggregate post view counters.

The first table will store one item per blog post, keyed by post slug. The
worker Lambda will update the item's view counter with an atomic increment for
each accepted post view event.

The initial item shape will be intentionally small:

```text
post_slug
view_count
updated_at
```

The table will use on-demand capacity so the blog does not need capacity
planning for a small and uneven analytics workload.

## Rationale

DynamoDB fits aggregate post views because the access pattern is simple:
increment a counter by post slug and read the current count by post slug.

Atomic updates avoid lost increments when multiple worker Lambda invocations
process events for the same post at the same time.

S3 and Athena are better for detailed event history and ad hoc analysis, but
they are the wrong serving path for a small counter shown by the application.
Querying Athena for current post counts would be slower and more operationally
awkward than reading one DynamoDB item.

On-demand capacity keeps the table simple to operate. Provisioned capacity can
be added later if traffic is steady enough to justify the extra tuning.

## Consequences

Aggregate counts are eventually consistent with accepted analytics events
because updates happen after SNS and SQS delivery.

Counts are approximate, not exact-once. Duplicate SQS delivery after a
successful DynamoDB update can increment the same post view more than once.
This matches ADR 0009 and is acceptable for first-pass blog analytics.

The simple key shape supports per-post counts. It does not support efficient
time-window analytics, referrer reports, or top-post queries by itself. Those
queries belong to the detailed S3 and Athena analytics path unless the
application later needs them online.

The DynamoDB table and IAM permissions are app-specific AWS resources and will
be managed by Terraform in this repository.
