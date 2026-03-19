"use client";

import Navbar from "@/components/Navbar";
import Hero from "@/components/Hero";
import Features from "@/components/Features";
import CodeDemo from "@/components/CodeDemo";
import CurrentScope from "@/components/CurrentScope";
import Footer from "@/components/Footer";

export default function Home() {
  return (
    <main className="min-h-screen bg-[#0a0a0f] overflow-x-hidden">
      <Navbar />
      <Hero />
      <CodeDemo />
      <Features />
      <CurrentScope />
      <Footer />
    </main>
  );
}
