# The data pipeline
The data pipeline takes data from S3 using S3 notifications,
filters for only the successful requests, then enriches the data with geoip data,
then pipes it to kinesis. From kinesis, we stream the data to an Elasticsearch cluster for now,
but this design allows for expansion into Salesforce and Mixpanel later on.
