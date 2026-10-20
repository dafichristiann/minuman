import React from "react";
import { CanKopiSusu, CanSodaLemon } from "./mascots/MascotCans";

export interface HeroBannerProps {
  onShopNowClick?: () => void;
}

export const HeroBanner: React.FC<HeroBannerProps> = ({ onShopNowClick }) => {
  const handleScroll = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    if (onShopNowClick) {
      onShopNowClick();
    } else {
      const target = document.getElementById("catalog-section");
      target?.scrollIntoView({ behavior: "smooth" });
    }
  };

  return (
    <section className="hero-banner" aria-labelledby="hero-heading">
      <div className="hero-banner__container">
        <div className="hero-banner__content">
          <div className="hero-banner__badge-row">
            <span className="badge badge--lemon">★ VINTAGE 1930s TASTE</span>
            <span className="badge badge--mint">KESEGARAN PREMIUM</span>
          </div>

          <h1 id="hero-heading" className="hero-banner__title">
            Sensasi Segar <span className="highlight-text">Kaleng Asli</span> Gaya 1930-an!
          </h1>

          <p className="hero-banner__lead">
            Minuman kaleng racikan bahan alami dengan cita rasa nostalgia klasik. Dari soda lemon segar, kopi susu gula aren legit, hingga teh melati wangi—siap meledakkan dahagamu!
          </p>

          <div className="hero-banner__benefits" role="list">
            <div className="benefit-pill" role="listitem">
              <span className="benefit-pill__icon" aria-hidden="true">🌱</span>
              <span className="benefit-pill__text">100% Gula Alami</span>
            </div>
            <div className="benefit-pill" role="listitem">
              <span className="benefit-pill__icon" aria-hidden="true">📦</span>
              <span className="benefit-pill__text">Pengiriman Se-Indonesia</span>
            </div>
            <div className="benefit-pill" role="listitem">
              <span className="benefit-pill__icon" aria-hidden="true">❄️</span>
              <span className="benefit-pill__text">Dingin Sampai Tempat</span>
            </div>
          </div>

          <div className="hero-banner__cta-group">
            <a
              href="#catalog-section"
              className="btn btn--primary btn--pill hero-cta-btn"
              onClick={handleScroll}
            >
              <span>Belanja Sekarang</span>
              <span aria-hidden="true">👇</span>
            </a>
            <span className="hero-banner__subnote">
              ⚡ Gratis ongkir s/d Rp20.000 dengan kode <strong>GRATISONGKIR</strong>
            </span>
          </div>
        </div>

        <div className="hero-banner__visual" aria-hidden="true">
          <div className="hero-mascot-stage">
            <div className="hero-mascot hero-mascot--left">
              <CanSodaLemon size={160} />
            </div>
            <div className="hero-mascot hero-mascot--right">
              <CanKopiSusu size={160} />
            </div>
            <div className="hero-banner__bubble">
              Slurrrp! Segar benerrr! 🥤
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
