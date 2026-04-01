import { SiteHeader } from "@/components/site-header";

export function Docs() {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <SiteHeader breadcrumbs={[{ label: "Docs", href: null }]} />
      <div className="flex min-h-0 flex-1 flex-col">
        <iframe
          title="Kinetic Docs"
          src="/docsify/index.html#/"
          className="h-full min-h-[720px] w-full flex-1 border-0 bg-white"
        />
      </div>
    </div>
  );
}