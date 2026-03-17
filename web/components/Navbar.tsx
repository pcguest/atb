"use client";

import { useState, useEffect } from "react";

export default function Navbar() {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => setScrolled(window.scrollY > 20);
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
        scrolled ? "bg-[#0a0a0f]/95 backdrop-blur-md border-b border-[#1e1e2e]" : "bg-transparent"
      }`}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center">
              <span className="font-mono font-bold text-white text-sm">A</span>
            </div>
            <span className="font-mono font-semibold text-white text-lg tracking-tight">ATB</span>
            <span className="hidden sm:inline-block text-[#6b7280] text-sm ml-1">
              / Agent Trace Bundle
            </span>
          </div>

          {/* Nav links */}
          <div className="hidden md:flex items-center gap-6">
            <a
              href="#features"
              className="text-[#9ca3af] hover:text-white text-sm transition-colors"
            >
              Features
            </a>
            <a href="#demo" className="text-[#9ca3af] hover:text-white text-sm transition-colors">
              Demo
            </a>
            <a
              href="#pricing"
              className="text-[#9ca3af] hover:text-white text-sm transition-colors"
            >
              Pricing
            </a>
            <a
              href="https://github.com/pcguest/atb"
              target="_blank"
              rel="noopener noreferrer"
              className="text-[#9ca3af] hover:text-white text-sm transition-colors flex items-center gap-1"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
              </svg>
              GitHub
            </a>
          </div>

          {/* CTA */}
          <div className="flex items-center gap-3">
            <a
              href="#waitlist"
              className="hidden sm:inline-flex items-center px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium transition-colors"
            >
              Join Waitlist
            </a>
          </div>
        </div>
      </div>
    </nav>
  );
}
