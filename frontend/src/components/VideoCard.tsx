import { Link } from "@tanstack/react-router";
import { API_URL, formatDuration, formatViews, type Video } from "@/lib/api";

export function VideoCard({ video, compact = false }: { video: Video; compact?: boolean }) {
  const processing = video.Progress !== undefined && video.Progress < 100;
  return (
    <Link
      to="/watch/$id"
      params={{ id: video.ID }}
      className={`group block overflow-hidden rounded-xl border border-transparent transition hover:border-primary/40 hover:scale-[1.02] ${compact ? "flex gap-3" : ""}`}
    >
      <div className={`relative ${compact ? "h-24 w-40 flex-shrink-0" : "aspect-video w-full"} overflow-hidden rounded-lg bg-muted`}>
        {video.ThumbnailUrl ? (
          <img src={`${API_URL}/videos/${video.ID}/thumbnail`} alt={video.Name} className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-xs text-muted-foreground">
            No thumbnail
          </div>
        )}
        {processing ? (
          <div className="absolute inset-0 flex items-center justify-center bg-black/70 text-xs font-medium uppercase tracking-wider text-primary">
            Processing…
          </div>
        ) : video.Duration > 0 ? (
          <span className="absolute bottom-2 right-2 rounded bg-black/80 px-1.5 py-0.5 text-xs font-medium text-white">
            {formatDuration(video.Duration)}
          </span>
        ) : null}
      </div>
      <div className={compact ? "min-w-0 flex-1" : "px-1 pt-3"}>
        <h3 className={`line-clamp-2 font-semibold leading-snug text-foreground ${compact ? "text-sm" : "text-base"}`}>
          {video.Name}
        </h3>
        <p className="mt-1 text-xs text-muted-foreground">
          {formatViews(video.ViewCount)} views
        </p>
      </div>
    </Link>
  );
}
