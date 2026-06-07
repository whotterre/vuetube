import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { API_URL, api, formatViews, type Video } from "@/lib/api";
import { VideoCard } from "@/components/VideoCard";
import { useAuth } from "@/lib/auth-context";

export const Route = createFileRoute("/watch/$id")({
  component: WatchPage,
});

function WatchPage() {
  const { id } = Route.useParams();
  const { isAuthenticated } = useAuth();
  const [video, setVideo] = useState<Video | null>(null);
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [liked, setLiked] = useState(false);
  const [likeCount, setLikeCount] = useState(0);
  const videoRef = useRef<HTMLVideoElement>(null);
  const playerRef = useRef<{ reset: () => void } | null>(null);

  // fetch video meta
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    setProcessing(false);
    api.getVideo(id)
      .then((v) => {
        if (cancelled) return;
        setVideo(v);
        setLikeCount(0); // initialize to 0; backend doesn't expose like_count on video
        document.title = `${v.Name} — VueTube`;
      })
      .catch((e) => !cancelled && setError(e.message))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
      document.title = "VueTube";
    };
  }, [id]);

  // init dash.js
  useEffect(() => {
    if (!video || !videoRef.current) return;
    let player: { reset: () => void } | null = null;
    let cancelled = false;

    const manifestUrl = `${API_URL}/videos/${id}/dash/manifest`;

    // Check status — 202 means still processing
    fetch(manifestUrl).then(async (res) => {
      if (cancelled) return;
      if (res.status === 202) {
        setProcessing(true);
        return;
      }
      if (!res.ok) {
        setError(`Manifest unavailable (${res.status})`);
        return;
      }
      const dashjs = await import("dashjs");
      if (cancelled || !videoRef.current) return;
      player = dashjs.MediaPlayer().create();
      // @ts-expect-error dashjs types
      player.initialize(videoRef.current, manifestUrl, false);
      playerRef.current = player;
    }).catch((e) => !cancelled && setError(e.message));

    return () => {
      cancelled = true;
      try { player?.reset(); } catch { /* noop */ }
      playerRef.current = null;
    };
  }, [id]);

  const { data: recommendations } = useQuery({
    queryKey: ["feed", id],
    queryFn: () => api.feed(id, 12),
    enabled: !!video,
  });

  const handleLike = async () => {
    if (!isAuthenticated) return;
    try {
      const res = await api.like(id);
      setLiked(res.liked);
      setLikeCount(res.like_count);
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <main className="relative z-10 mx-auto max-w-[1600px] px-6 py-8">
      <div className="grid gap-10 lg:grid-cols-[minmax(0,7fr)_minmax(0,3fr)]">
        <div>
          <div className="aspect-video w-full overflow-hidden rounded-xl bg-black ring-1 ring-border">
            {processing ? (
              <div className="flex h-full w-full flex-col items-center justify-center text-center">
                <div className="font-mono text-[11px] uppercase tracking-[0.3em] text-primary">◐ Processing</div>
                <p className="mt-3 max-w-md font-display text-2xl italic text-foreground">
                  Still developing in the darkroom.
                </p>
                <p className="mt-1 text-sm text-muted-foreground">Check back in a moment.</p>
              </div>
            ) : error ? (
              <div className="flex h-full w-full items-center justify-center text-sm text-muted-foreground">
                {error}
              </div>
            ) : (
              <video ref={videoRef} controls className="h-full w-full bg-black" />
            )}
          </div>

          {loading ? (
            <div className="mt-6 h-10 w-2/3 animate-pulse rounded bg-muted" />
          ) : video ? (
            <div className="mt-6 border-b border-border pb-6">
              <h1 className="font-display text-4xl italic leading-tight md:text-5xl">{video.Name}</h1>
              <div className="mt-4 flex items-center justify-between gap-4">
                <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-muted-foreground">
                  {formatViews(video.ViewCount)} views · {video.Resolution || "—"}
                </p>
                <button
                  onClick={handleLike}
                  disabled={!isAuthenticated}
                  className={`group flex items-center gap-2 rounded-full border px-5 py-2 text-sm font-medium transition ${
                    liked
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-border bg-card text-foreground hover:border-primary/60 hover:text-primary"
                  } ${!isAuthenticated ? "cursor-not-allowed opacity-60" : ""}`}
                  title={isAuthenticated ? "Like" : "Sign in to like"}
                >
                  <span className={liked ? "" : "transition group-hover:scale-110"}>♥</span>
                  <span className="font-mono text-xs">{likeCount}</span>
                </button>
              </div>
            </div>
          ) : null}
        </div>

        <aside>
          <p className="mb-5 font-mono text-[11px] uppercase tracking-[0.25em] text-primary">
            ╱╱ Up next
          </p>
          <div className="flex flex-col gap-5">
            {(recommendations ?? []).filter((r) => r.ID !== id).map((r) => (
              <VideoCard key={r.ID} video={r} compact />
            ))}
            {recommendations && recommendations.length === 0 && (
              <Link to="/" className="text-sm text-muted-foreground hover:text-foreground">
                No recommendations — browse the feed →
              </Link>
            )}
          </div>
        </aside>
      </div>
    </main>
  );
}
