import { createFileRoute, useNavigate, Link } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { api, type Video } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

export const Route = createFileRoute("/upload")({
  component: UploadPage,
});

function UploadPage() {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const [file, setFile] = useState<File | null>(null);
  const [title, setTitle] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const [uploadPct, setUploadPct] = useState(0);
  const [uploading, setUploading] = useState(false);
  const [processingPct, setProcessingPct] = useState<number | null>(null);
  const [createdId, setCreatedId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, []);

  if (!isAuthenticated) {
    return (
      <main className="mx-auto max-w-md px-6 py-24 text-center">
        <h1 className="text-2xl font-bold">Sign in to upload</h1>
        <p className="mt-2 text-sm text-muted-foreground">You need an account to share videos on VueTube.</p>
        <div className="mt-6 flex justify-center gap-3">
          <Link to="/login" className="rounded-md border border-border px-4 py-2 text-sm font-medium hover:bg-muted">Sign in</Link>
          <Link to="/signup" className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground">Sign up</Link>
        </div>
      </main>
    );
  }

  const startUpload = async () => {
    if (!file || !title.trim()) return;
    setError(null);
    setUploading(true);
    setUploadPct(0);
    try {
      const video: Video = await api.upload(file, title.trim(), setUploadPct);
      setCreatedId(video.ID);
      setProcessingPct(video.Progress ?? 0);
      pollRef.current = setInterval(async () => {
        try {
          const v = await api.getVideo(video.ID);
          setProcessingPct(v.Progress);
          if (v.Progress >= 100) {
            if (pollRef.current) clearInterval(pollRef.current);
          }
        } catch (e) {
          console.error(e);
        }
      }, 3000);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setUploading(false);
    }
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const f = e.dataTransfer.files?.[0];
    if (f && f.type.startsWith("video/")) setFile(f);
  };

  const done = processingPct !== null && processingPct >= 100;

  return (
    <main className="mx-auto max-w-2xl px-6 py-12">
      <h1 className="text-3xl font-bold tracking-tight">Upload a video</h1>
      <p className="mt-1 text-sm text-muted-foreground">Up to 500MB. We'll handle adaptive streaming for you.</p>

      {!createdId ? (
        <div className="mt-8 space-y-6">
          <label
            onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
            onDragLeave={() => setDragOver(false)}
            onDrop={onDrop}
            className={`flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed py-16 text-center transition ${
              dragOver ? "border-primary bg-primary/5" : "border-border bg-card"
            }`}
          >
            <input
              type="file"
              accept="video/*"
              className="hidden"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
            <div className="text-lg font-semibold">{file ? file.name : "Drop a video here"}</div>
            <p className="mt-1 text-sm text-muted-foreground">
              {file ? `${(file.size / (1024 * 1024)).toFixed(1)} MB` : "or click to browse"}
            </p>
          </label>

          <div>
            <label className="block text-sm font-medium">Title</label>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Give your video a title"
              className="mt-2 w-full rounded-md border border-border bg-input px-3 py-2.5 text-sm outline-none focus:border-primary"
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <button
            disabled={!file || !title.trim() || uploading}
            onClick={startUpload}
            className="w-full rounded-md bg-primary py-3 text-sm font-semibold text-primary-foreground transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {uploading ? `Uploading… ${uploadPct}%` : "Upload"}
          </button>
          {uploading && (
            <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
              <div className="h-full bg-primary transition-all" style={{ width: `${uploadPct}%` }} />
            </div>
          )}
        </div>
      ) : (
        <div className="mt-8 rounded-xl border border-border bg-card p-8 text-center">
          {done ? (
            <>
              <div className="text-sm font-medium uppercase tracking-widest text-primary">Ready</div>
              <h2 className="mt-2 text-xl font-bold">Your video is live</h2>
              <button
                onClick={() => navigate({ to: "/watch/$id", params: { id: createdId } })}
                className="mt-6 rounded-md bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground"
              >
                Watch now
              </button>
            </>
          ) : (
            <>
              <div className="text-sm font-medium uppercase tracking-widest text-primary">Processing</div>
              <h2 className="mt-2 text-xl font-bold">Preparing adaptive streams</h2>
              <p className="mt-2 text-sm text-muted-foreground">
                You can leave this page — we'll keep going in the background.
              </p>
              <div className="mt-6 h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full bg-primary transition-all"
                  style={{ width: `${processingPct ?? 0}%` }}
                />
              </div>
              <p className="mt-2 text-xs text-muted-foreground">{processingPct ?? 0}%</p>
            </>
          )}
        </div>
      )}
    </main>
  );
}
