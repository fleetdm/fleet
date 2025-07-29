package common_mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractRDSRegion(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
		wantErr  bool
	}{
		{
			name:     "Aurora Serverless v2 cluster endpoint",
			endpoint: "fleet-aurora-serverless-v2.cluster-chokegk86qom.us-east-2.rds.amazonaws.com",
			want:     "us-east-2",
			wantErr:  false,
		},
		{
			name:     "Aurora standard cluster endpoint",
			endpoint: "fleet-aurora-standard.cluster-abc123def456.us-west-2.rds.amazonaws.com",
			want:     "us-west-2",
			wantErr:  false,
		},
		{
			name:     "Aurora read-only cluster endpoint",
			endpoint: "fleet-aurora-cluster.cluster-ro-xyz789ghi012.eu-west-1.rds.amazonaws.com",
			want:     "eu-west-1",
			wantErr:  false,
		},
		{
			name:     "RDS instance endpoint",
			endpoint: "database-1.cde345fgh678.ap-southeast-1.rds.amazonaws.com",
			want:     "ap-southeast-1",
			wantErr:  false,
		},
		{
			name:     "RDS MariaDB instance",
			endpoint: "fleet-rds-mariadb.chokegk86qom.us-east-2.rds.amazonaws.com",
			want:     "us-east-2",
			wantErr:  false,
		},
		{
			name:     "RDS proxy endpoint",
			endpoint: "my-proxy.proxy-abc123def456.us-east-1.rds.amazonaws.com",
			want:     "us-east-1",
			wantErr:  false,
		},
		{
			name:     "China region endpoint",
			endpoint: "myinstance.abc123.cn-north-1.rds.amazonaws.com.cn",
			want:     "cn-north-1",
			wantErr:  false,
		},
		{
			name:     "China region cluster",
			endpoint: "mycluster.cluster-abc123.cn-northwest-1.rds.amazonaws.com.cn",
			want:     "cn-northwest-1",
			wantErr:  false,
		},
		{
			name:     "GovCloud region",
			endpoint: "database.abc123.us-gov-west-1.rds.amazonaws.com",
			want:     "us-gov-west-1",
			wantErr:  false,
		},
		{
			name:     "Invalid endpoint - not RDS",
			endpoint: "not-an-rds-endpoint.example.com",
			wantErr:  true,
		},
		{
			name:     "Invalid endpoint - too short",
			endpoint: "short.com",
			wantErr:  true,
		},
		{
			name:     "Invalid endpoint - no region",
			endpoint: "instance.rds.amazonaws.com",
			wantErr:  true,
		},
		{
			name:     "Empty endpoint",
			endpoint: "",
			wantErr:  true,
		},
		{
			name:     "Endpoint with port should fail",
			endpoint: "mydb.abc123.us-east-1.rds.amazonaws.com:3306",
			wantErr:  true,
		},
		{
			name:     "Complex instance name with hyphens",
			endpoint: "my-complex-db-name.xyz789.sa-east-1.rds.amazonaws.com",
			want:     "sa-east-1",
			wantErr:  false,
		},
		{
			name:     "Aurora global database endpoint",
			endpoint: "global-database-1.cluster-cust-abc123.us-east-1.rds.amazonaws.com",
			want:     "us-east-1",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractRDSRegion(tt.endpoint)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "RDS endpoint")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
