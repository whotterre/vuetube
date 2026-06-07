import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Play, Pause, Volume2, VolumeX, Maximize, Minimize, ThumbsUp, ArrowLeft } from "lucide-react";
import { API_URL, api, formatViews, type Video } from "@/lib/api";
import { VideoCard } from "@/components/VideoCard";
import { useAuth } from "@/lib/auth-context";

export const Route = createFileRoute("/watch/$id")({
  component: WatchPage,
});

function WatchPage() {
  const { id } = Route.useParams();
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate({ to: "/login", replace: true });
    }
  }, [isAuthenticated, navigate]);

  const [video, setVideo] = useState<Video | null>(null);
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [liked, setLiked] = useState(false);
  const [likeCount, setLikeCount] = useState(0);
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const playerRef = useRef<{ reset: () => void } | null>(null);

  // Video player state
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isBuffering, setIsBuffering] = useState(true);

  const togglePlay = () => {
    if (videoRef.current) {
      if (videoRef.current.paused) videoRef.current.play();
      else videoRef.current.pause();
    }
  };

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const time = parseFloat(e.target.value);
    if (videoRef.current) {
      videoRef.current.currentTime = time;
      setCurrentTime(time);
    }
  };

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const vol = parseFloat(e.target.value);
    if (videoRef.current) {
      videoRef.current.volume = vol;
      setVolume(vol);
      setIsMuted(vol === 0);
    }
  };

  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !videoRef.current.muted;
      setIsMuted(videoRef.current.muted);
    }
  };

  const toggleFullscreen = () => {
    if (!containerRef.current) return;
    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen().catch(console.error);
    } else {
      document.exitFullscreen();
    }
  };

  useEffect(() => {
    const handleFullscreenChange = () => setIsFullscreen(!!document.fullscreenElement);
    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", handleFullscreenChange);
  }, []);

  const formatTime = (time: number) => {
    if (isNaN(time)) return "0:00";
    const m = Math.floor(time / 60);
    const s = Math.floor(time % 60);
    return `${m}:${s.toString().padStart(2, "0")}`;
  };

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
        setLikeCount(0); 
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
    if (!videoRef.current) return;
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

  if (!isAuthenticated) return null;

  return (
    <main className="relative z-10 mx-auto max-w-[1600px] px-6 py-8">
      <div className="mb-6">
        <button onClick={() => window.history.back()} className="flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors w-fit">
          <ArrowLeft className="h-4 w-4" />
          Back
        </button>
      </div>
      <div className="flex flex-col lg:flex-row gap-10">
        <div className="w-full lg:w-[50%] xl:w-[55%]">
          <div ref={containerRef} className="group relative aspect-video w-full overflow-hidden rounded-xl bg-black ring-1 ring-border">
            {processing ? (
              <div className="flex h-full w-full flex-col items-center justify-center text-center">
                <div className="font-mono text-[11px] uppercase tracking-[0.3em] text-primary">◐ Processing</div>
                <p className="mt-3 max-w-md font-display text-2xl text-foreground">
                  Still developing in the darkroom.
                </p>
                <p className="mt-1 text-sm text-muted-foreground">Check back in a moment.</p>
              </div>
            ) : error ? (
              <div className="flex h-full w-full items-center justify-center text-sm text-muted-foreground">
                {error}
              </div>
            ) : (
              <>
                <video
                  ref={videoRef}
                  className="h-full w-full bg-black cursor-pointer"
                  onClick={togglePlay}
                  onTimeUpdate={(e) => setCurrentTime(e.currentTarget.currentTime)}
                  onLoadedMetadata={(e) => setDuration(e.currentTarget.duration)}
                  onPlay={() => setIsPlaying(true)}
                  onPause={() => setIsPlaying(false)}
                  onWaiting={() => setIsBuffering(true)}
                  onPlaying={() => setIsBuffering(false)}
                  onCanPlay={() => setIsBuffering(false)}
                />
                
                {/* Custom Buffering Loader */}
                {isBuffering && (
                  <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/40 backdrop-blur-[2px] transition-opacity">
                    <div className="relative flex h-16 w-16 items-center justify-center">
                      <div className="absolute inset-0 animate-[spin_1.5s_linear_infinite] rounded-full border-[3px] border-transparent border-t-primary border-r-primary/50"></div>
                      <div className="absolute inset-2 animate-[spin_2s_linear_infinite_reverse] rounded-full border-[3px] border-transparent border-b-primary border-l-primary/30"></div>
                      <div className="h-3 w-3 animate-pulse rounded-full bg-primary"></div>
                    </div>
                  </div>
                )}
                
                {/* Top Controls Bar */}
                <div className="absolute top-0 left-0 right-0 bg-gradient-to-b from-black/80 to-transparent px-5 py-4 flex items-start justify-end opacity-0 transition-opacity duration-300 group-hover:opacity-100">
                  <button
                    onClick={(e) => { e.stopPropagation(); handleLike(); }}
                    disabled={!isAuthenticated}
                    className={`flex items-center gap-2.5 rounded-full px-5 py-2.5 text-base font-medium transition backdrop-blur ${
                      liked
                        ? "bg-primary text-primary-foreground"
                        : "bg-black/40 text-white hover:bg-white/20"
                    } ${!isAuthenticated ? "cursor-not-allowed opacity-60" : ""}`}
                    title={isAuthenticated ? "Like" : "Sign in to like"}
                  >
                    <ThumbsUp className={`h-5 w-5 ${liked ? "fill-current" : ""}`} />
                    <span className="font-mono text-sm">{likeCount}</span>
                  </button>
                </div>
                
                {/* Big Play Button Overlay */}
                {!isPlaying && (
                  <div 
                    className="absolute inset-0 flex items-center justify-center bg-black/20 cursor-pointer transition-opacity"
                    onClick={togglePlay}
                  >
                    <div className="flex h-24 w-24 items-center justify-center rounded-full bg-primary/90 text-primary-foreground backdrop-blur transition-transform hover:scale-110">
                      <Play className="h-12 w-12 ml-2" fill="currentColor" />
                    </div>
                  </div>
                )}

                {/* Bottom Controls Bar */}
                <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/95 via-black/60 to-transparent px-5 py-4 opacity-0 transition-opacity duration-300 group-hover:opacity-100">
                  <div className="flex items-center gap-3">
                    <span className="text-xs font-mono text-white/90 w-10 text-right">{formatTime(currentTime)}</span>
                    <input
                      type="range"
                      min="0"
                      max={duration || 100}
                      value={currentTime}
                      onChange={handleSeek}
                      className="flex-1 cursor-pointer accent-primary h-1.5 bg-white/30 rounded-full"
                    />
                    <span className="text-xs font-mono text-white/90 w-10">{formatTime(duration)}</span>
                  </div>

                  <div className="mt-4 flex items-center justify-between text-white">
                    <div className="flex items-center gap-6">
                      <button onClick={togglePlay} className="text-white hover:text-primary transition-colors">
                        {isPlaying ? <Pause className="h-7 w-7" fill="currentColor" /> : <Play className="h-7 w-7" fill="currentColor" />}
                      </button>

                      <div className="group/volume flex items-center gap-2">
                        <button onClick={toggleMute} className="text-white hover:text-primary transition-colors">
                          {isMuted || volume === 0 ? <VolumeX className="h-5 w-5" /> : <Volume2 className="h-5 w-5" />}
                        </button>
                        <input
                          type="range"
                          min="0"
                          max="1"
                          step="0.05"
                          value={isMuted ? 0 : volume}
                          onChange={handleVolumeChange}
                          className="w-0 opacity-0 transition-all duration-300 ease-in-out group-hover/volume:w-20 group-hover/volume:opacity-100 cursor-pointer accent-white h-1.5 bg-white/30 rounded-full"
                        />
                      </div>
                    </div>

                    <button onClick={toggleFullscreen} className="text-white hover:text-primary transition-colors">
                      {isFullscreen ? <Minimize className="h-5 w-5" /> : <Maximize className="h-5 w-5" />}
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>

          {loading ? (
            <div className="mt-6 h-10 w-2/3 animate-pulse rounded bg-muted" />
          ) : video ? (
            <div className="mt-6 flex flex-col justify-between gap-4 pb-6 md:flex-row md:items-end">
              <h1 className="font-display text-2xl leading-tight md:text-3xl lg:max-w-[65%]">{video.Name}</h1>
              <div className="flex shrink-0 flex-wrap items-center gap-3 lg:justify-end lg:pb-1">
                <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-muted-foreground">
                  {formatViews(video.ViewCount)} views
                </p>
                <span className="text-muted-foreground/30">•</span>
                <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-muted-foreground">
                  {video.Resolution || "—"}
                </p>
                {video.UploadedAt && (
                  <>
                    <span className="text-muted-foreground/30">•</span>
                    <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-muted-foreground">
                      {new Date(video.UploadedAt).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })}
                    </p>
                  </>
                )}
              </div>
            </div>
          ) : null}
        </div>

        <aside className="w-full flex-1 lg:w-[50%] xl:w-[45%]">
          <p className="mb-5 font-mono text-[11px] uppercase tracking-[0.25em] text-primary">
            You Might Like
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
