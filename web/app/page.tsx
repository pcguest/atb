"use client";

import Navbar from "@/components/Navbar";
import Hero from "@/components/Hero";
import Features from "@/components/Features";
import CodeDemo from "@/components/CodeDemo";
import CurrentScope from "@/components/CurrentScope";
import Footer from "@/components/Footer";

export default function Home() {
  return (
    <main className="min-h-screen overflow-x-hidden bg-background">
      <Navbar />
      <Hero />
      <Features />
      <CodeDemo />
      <CurrentScope />
      <Footer />
    </main>
  );
}
