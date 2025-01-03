package common_mysql

// BatchProcessSimple is a simple utility function to batch process a slice of payloads.
// Provide a slice of payloads, a batch size, and a function to execute on each batch.
func BatchProcessSimple[T any](
	payloads []T,
	batchSize int,
	executeBatch func(payloadsInThisBatch []T) error,
) error {
	if len(payloads) == 0 || batchSize <= 0 || executeBatch == nil {
		return nil
	}

	for i := 0; i < len(payloads); i += batchSize {
		start := i
		end := i + batchSize
		if end > len(payloads) {
			end = len(payloads)
		}
		if err := executeBatch(payloads[start:end]); err != nil {
			return err
		}
	}

	return nil
}
