# File Carving with Fleet

### Carve Block Size

The `carver_block_size` flag should be configured in osquery.

How to choose the correct value? 2MB (2000000) is a good starting value.

The configured value must be less than the value of `max_allowed_packet` in the MySQL connection, allowing for some overhead. The default for MySQL 5.7 is 4MB and for MySQL 8 it is 64MB.

Using a smaller value for `carver_block_size` will lead to more HTTP requests during the carving process, resulting in longer carve times and higher load on the Fleet server. If the value is too high, HTTP requests may run long enough to cause server timeouts.


### Troubleshooting

#### Ensure  `carver_block_size` is set appropriately

This value must be less than the `max_allowed_packet` setting in MySQL. If it is too large, MySQL will reject the writes.

The value must be small enough that HTTP requests do not time out.

