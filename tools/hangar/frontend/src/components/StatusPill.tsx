// StatusPill mirrors ServerTab's HealthChip: a rounded pill with a status dot,
// a label, and an UPPERCASE state tag, tinted by up/down. Shared by the SCEP,
// MDM assets, and TUF tabs so status reads the same everywhere.
export function StatusPill({
  label,
  up,
  upText = "up",
  downText = "down",
}: {
  label: string;
  up: boolean;
  upText?: string;
  downText?: string;
}) {
  const color = up ? "var(--core-fleet-green)" : "var(--ui-error)";
  const bg = up ? "var(--tint-success-soft)" : "var(--tint-danger-soft)";
  return (
    <div
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
        padding: "3px 10px 3px 8px",
        background: bg,
        border: `1px solid ${color}`,
        borderRadius: 999,
        fontSize: "var(--fs-xx-small)",
        color,
      }}
    >
      <span className={`dot ${up ? "run" : "fail"}`} />
      <span style={{ fontWeight: 600 }}>{label}</span>
      <span
        style={{
          fontSize: "var(--fs-xxx-small)",
          textTransform: "uppercase",
          letterSpacing: "0.05em",
          color: "var(--core-fleet-white)",
          background: color,
          padding: "1px 5px",
          borderRadius: 3,
        }}
      >
        {up ? upText : downText}
      </span>
    </div>
  );
}
