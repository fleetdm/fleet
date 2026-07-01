export function Stub({ name }: { name: string }) {
  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        flexDirection: "column",
        gap: 8,
        color: "var(--app-text-dim)",
      }}
    >
      <div style={{ fontSize: "var(--fs-medium)", fontWeight: 600 }}>
        {name}
      </div>
      <div style={{ fontSize: "var(--fs-xx-small)" }}>
        Coming in a future iteration
      </div>
    </div>
  );
}
