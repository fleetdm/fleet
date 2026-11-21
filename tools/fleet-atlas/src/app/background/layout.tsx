import BackgroundTabs from "@/components/background/Tabs";

export default function BackgroundLayout({
  children,
}: LayoutProps<"/background">) {
  return (
    <div className="mx-auto max-w-5xl px-10 py-10">
      <header className="mb-6 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Background work
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Fleet does a lot off the HTTP request path. Two mechanisms:{" "}
          <strong>cron schedules</strong> (recurring, lock-coordinated via
          Redis) and the <strong>worker queue</strong> (durable jobs in MySQL).
        </p>
      </header>
      <BackgroundTabs />
      {children}
    </div>
  );
}
