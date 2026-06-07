import { createFileRoute, Link, useSearch } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { VideoCard } from "@/components/VideoCard";
import { useAuth } from "@/lib/auth-context";

export const Route = createFileRoute("/")({
  validateSearch: (search: Record<string, unknown>): { q?: string } => {
    return {
      q: search.q as string | undefined,
    };
  },
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
  const { isAuthenticated } = useAuth();
  const search = useSearch({ from: "/" });
  const q = search.q?.toLowerCase();

  const { data, isLoading, error } = useQuery({
    queryKey: ["feed"],
    queryFn: () => api.feed(undefined, 24),
  });

  const filteredData = data
    ? data.filter((v) => !q || v.Name.toLowerCase().includes(q) || (v.Resolution && v.Resolution.toLowerCase().includes(q)))
    : [];

  return (
    <main className="relative z-10 mx-auto max-w-[1600px] px-6 py-12">
      {!isAuthenticated && (
        <div className="mb-12 pb-8">
          <h1 className="font-display text-6xl leading-[0.95] md:text-7xl">
            What's <span className="text-primary">streaming</span><br />right now.
          </h1>
          <p className="mt-4 max-w-md text-sm leading-relaxed text-muted-foreground">
            An independent feed of moving images, served adaptively at whatever bitrate your line can carry.
          </p>
        </div>
      )}

      {/* UX Heading */}
      {(!isLoading && !error && filteredData && filteredData.length > 0) && (
        <div className="mb-8 flex items-baseline justify-between border-b border-border pb-4">
          <h2 className="font-display text-2xl md:text-3xl">
            {q ? `Search Results (${filteredData.length})` : "Videos"}
          </h2>
        </div>
      )}

      {isLoading ? (
        <SkeletonGrid />
      ) : error ? (
        <EmptyState title="Couldn't load videos" message={(error as Error).message} />
      ) : !filteredData || filteredData.length === 0 ? (
        <EmptyState title={q ? `No videos found for "${q}"` : "No videos yet"} message={q ? "Try a different search term." : "Be the first to upload something."} />
      ) : (
        <div className="grid gap-x-6 gap-y-12 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))]">
          {filteredData.slice(0, isAuthenticated ? undefined : 4).map((v) => <VideoCard key={v.ID} video={v} />)}
        </div>
      )}
    </main>
  );
}

function SkeletonGrid() {
  const { isAuthenticated } = useAuth();
  return (
    <div className="grid gap-x-5 gap-y-10 [grid-template-columns:repeat(auto-fill,minmax(280px,1fr))]">
      {Array.from({ length: isAuthenticated ? 8 : 4 }).map((_, i) => (
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
