package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (ds *Datastore) BatchUpsertCertificateTemplates(ctx context.Context, certificates []*fleet.Certificate) error {
	if len(certificates) == 0 {
		return nil
	}

	const argsCountInsertCertificate = 4

	const sqlInsertCertificate = `
		INSERT INTO certificate_templates (
			name,
			team_id,
			certificate_authority_id,
			subject_name
		) VALUES %s
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			team_id = VALUES(team_id),
			certificate_authority_id = VALUES(certificate_authority_id),
			subject_name = VALUES(subject_name)
	`

	var placeholders strings.Builder
	args := make([]interface{}, 0, len(certificates)*argsCountInsertCertificate)

	for _, cert := range certificates {
		args = append(args, cert.Name, cert.TeamID, cert.CertificateAuthorityID, cert.SubjectName)
		placeholders.WriteString("(?,?,?,?),")
	}

	stmt := fmt.Sprintf(sqlInsertCertificate, strings.TrimSuffix(placeholders.String(), ","))

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "upserting certificate_templates")
	}

	return nil
}
