resource "aws_glue_catalog_database" "analytics" {
  name = "mrembiasz_blog_analytics"
}

resource "aws_glue_catalog_table" "raw_analytics_events" {
  name          = "raw_events"
  database_name = aws_glue_catalog_database.analytics.name
  table_type    = "EXTERNAL_TABLE"

  parameters = {
    EXTERNAL                    = "TRUE"
    classification              = "parquet"
    "projection.enabled"        = "true"
    "projection.year.type"      = "integer"
    "projection.year.range"     = "2026,2100"
    "projection.year.digits"    = "4"
    "projection.month.type"     = "integer"
    "projection.month.range"    = "1,12"
    "projection.month.digits"   = "2"
    "projection.day.type"       = "integer"
    "projection.day.range"      = "1,31"
    "projection.day.digits"     = "2"
    "storage.location.template" = "s3://${aws_s3_bucket.raw_analytics.bucket}/raw/year=$${year}/month=$${month}/day=$${day}/"
  }

  partition_keys {
    name = "year"
    type = "int"
  }

  partition_keys {
    name = "month"
    type = "int"
  }

  partition_keys {
    name = "day"
    type = "int"
  }

  storage_descriptor {
    location      = "s3://${aws_s3_bucket.raw_analytics.bucket}/raw/"
    input_format  = "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat"
    output_format = "org.apache.hadoop.hive.ql.io.parquet.MapredParquetOutputFormat"

    ser_de_info {
      serialization_library = "org.apache.hadoop.hive.ql.io.parquet.serde.ParquetHiveSerDe"
    }

    columns {
      name = "event_type"
      type = "string"
    }

    columns {
      name = "post_slug"
      type = "string"
    }

    columns {
      name = "received_at"
      type = "string"
    }

    columns {
      name = "client_ip"
      type = "string"
    }

    columns {
      name = "user_agent"
      type = "string"
    }

    columns {
      name = "referer"
      type = "string"
    }
  }
}

resource "aws_kinesis_firehose_delivery_stream" "raw_analytics" {
  name        = "mrembiasz-blog-raw-analytics"
  destination = "extended_s3"
  tags        = local.tags
  depends_on  = [aws_iam_role_policy_attachment.raw_analytics_firehose]

  extended_s3_configuration {
    role_arn           = aws_iam_role.raw_analytics_firehose.arn
    bucket_arn         = aws_s3_bucket.raw_analytics.arn
    buffering_interval = 300
    buffering_size     = 64
    prefix             = "raw/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
    error_output_prefix = join("/", [
      "errors/!{firehose:error-output-type}",
      "year=!{timestamp:yyyy}",
      "month=!{timestamp:MM}",
      "day=!{timestamp:dd}/",
    ])

    data_format_conversion_configuration {
      enabled = true

      input_format_configuration {
        deserializer {
          open_x_json_ser_de {}
        }
      }

      output_format_configuration {
        serializer {
          parquet_ser_de {
            compression = "SNAPPY"
          }
        }
      }

      schema_configuration {
        database_name = aws_glue_catalog_database.analytics.name
        region        = "eu-central-1"
        role_arn      = aws_iam_role.raw_analytics_firehose.arn
        table_name    = aws_glue_catalog_table.raw_analytics_events.name
        version_id    = "LATEST"
      }
    }
  }
}
