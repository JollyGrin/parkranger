import "./style.css";

document.addEventListener("DOMContentLoaded", () => {
  const playerEl = document.getElementById("demo-player");
  if (playerEl && window.AsciinemaPlayer) {
    AsciinemaPlayer.create("demo.cast", playerEl, {
      autoPlay: true,
      loop: true,
      idleTimeLimit: 2,
      poster: "npt:0:3",
      fit: "width",
      terminalFontFamily:
        '"JetBrains Mono", "Fira Code", "SF Mono", monospace',
    });
  }

  // Smooth reveal on scroll
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add("visible");
        }
      });
    },
    { threshold: 0.1 },
  );

  document.querySelectorAll(".fade-in").forEach((el) => observer.observe(el));
});
