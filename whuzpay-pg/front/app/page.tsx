import Navbar from "./components/landing/Navbar";
import Hero from "./components/landing/Hero";
import LogoStrip from "./components/landing/LogoStrip";
import ProductShowcase from "./components/landing/ProductShowcase";
import HowItWorks from "./components/landing/HowItWorks";
import Features from "./components/landing/Features";
import TestimonialCarousel from "./components/landing/TestimonialCarousel";
import FAQSection from "./components/landing/FAQSection";
import CtaRow from "./components/landing/CtaRow";
import Footer from "./components/landing/Footer";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col bg-white">
      <Navbar />
      <main className="flex-1">
        <Hero />
        <LogoStrip />
        <ProductShowcase />
        <HowItWorks />
        <Features />
        <TestimonialCarousel />
        <FAQSection />
        <CtaRow />
      </main>
      <Footer />
    </div>
  );
}
