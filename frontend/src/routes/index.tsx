import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { VideoCard } from "@/components/VideoCard";

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "VueTube — Watch the independent web" },
      { name: "description", content: "A modern video streaming platform with adaptive DASH playback." },
      { property: "og:title", content: "VueTube" },
      { property: "og:description", content: "A modern video streaming platform with adaptive DASH playback." },
    ],
  }),
  component: Home,
});

function Home() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["feed"],
    queryFn: () => api.feed(undefined, 24),
  });

  return (
    <main className="relative z-10 mx-auto max-w-[1600px] px-6 py-12">
      <div className="mb-12 flex items-end justify-between border-b border-border pb-8">
        <div>
          <p className="mb-3 font-mono text-[11px] uppercase tracking-[0.25em] text-primary">
            ╱╱ Today's edition
          </p>
          <h1 className="font-display text-6xl italic leading-[0.95] md:text-7xl">
            What's <span className="text-primary">streaming</span><br />right now.
          </h1>
          <p className="mt-4 max-w-md text-sm leading-relaxed text-muted-foreground">
            An independent feed of moving images, served adaptively at whatever bitrate your line can carry.
          </p>
        </div>
        <Link
          to="/upload"
          className="hidden rounded-full bg-foreground px-5 py-2.5 text-sm font-medium text-background transition hover:bg-primary hover:text-primary-foreground md:inline-flex"
        >
          + Contribute a film
        </Link>
      </div>

      {isLoading ? (
        <SkeletonGrid />
      ) : error ? (
        <EmptyState title="Couldn't load videos" message={(error as Error).message} />
      ) : !data || data.length === 0 ? (
        <EmptyState title="No videos yet" message="Be the first to upload something." />
      ) : (
        <div className="grid gap-x-6 gap-y-12 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))]">
          {data.map((v) => <VideoCard key={v.ID} video={v} />)}
        </div>
      )}
    </main>
  );
}

function SkeletonGrid() {
  return (
    <div className="grid gap-x-5 gap-y-10 [grid-template-columns:repeat(auto-fill,minmax(280px,1fr))]">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i}>
          <div className="aspect-video w-full animate-pulse rounded-lg bg-muted" />
          <div className="mt-3 h-4 w-3/4 animate-pulse rounded bg-muted" />
          <div className="mt-2 h-3 w-1/3 animate-pulse rounded bg-muted" />
        </div>
      ))}
    </div>
  );
}

function EmptyState({ title, message }: { title: string; message: string }) {
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-24 text-center">
      <h2 className="text-lg font-semibold">{title}</h2>
      <p className="mt-2 max-w-md text-sm text-muted-foreground">{message}</p>
    </div>
  );
}
