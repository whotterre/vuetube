import { Link, useNavigate } from "@tanstack/react-router";
import { Search } from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useState } from "react";

export function Navbar() {
  const { isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState("");

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.trim()) {
      navigate({ to: "/", search: { q: searchQuery.trim() } });
    }
  };

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border bg-background/85 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-[1600px] items-center justify-between px-6">
        <Link to="/" className="group flex items-center gap-2 hover:opacity-80 transition-opacity shrink-0" aria-label="VueTube Home">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-8 w-8 text-primary"
          >
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
            <polygon points="10 8 16 12 10 16 10 8" fill="currentColor" />
          </svg>
         <p className="font-display text-xl font-medium tracking-tight">Vue<span className="text-primary font-bold">Tube</span></p>
        </Link>
        
        <form onSubmit={handleSearch} className="hidden md:flex flex-1 items-center justify-center max-w-lg mx-8">
          <div className="relative w-full">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search..."
              className="w-full rounded-full border border-border bg-black/20 py-2.5 pl-11 pr-4 text-sm outline-none transition focus:border-primary focus:bg-background focus:ring-1 focus:ring-primary/50 placeholder:text-muted-foreground/70"
            />
          </div>
        </form>

        <nav className="flex items-center gap-2 shrink-0">
          {isAuthenticated ? (
            <>
              <Link
                to="/upload"
                className="rounded-full bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition hover:brightness-110"
              >
                Upload
              </Link>
              <button
                onClick={() => { logout(); navigate({ to: "/" }); }}
                className="rounded-full border border-border px-4 py-2 text-sm font-medium text-muted-foreground transition hover:border-foreground/30 hover:text-foreground"
              >
                Sign out
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="rounded-full px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground">
                Sign in
              </Link>
              <Link
                to="/signup"
                className="rounded-full bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition hover:brightness-110"
              >
                Sign up
              </Link>
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
