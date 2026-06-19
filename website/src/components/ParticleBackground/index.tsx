import {useEffect, useRef} from 'react';
import type {ReactNode} from 'react';
import styles from './styles.module.css';

interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  size: number;
  opacity: number;
}

export default function ParticleBackground(): ReactNode {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Set canvas size
    const resizeCanvas = () => {
      canvas.width = window.innerWidth;
      canvas.height = window.innerHeight;
    };
    resizeCanvas();
    window.addEventListener('resize', resizeCanvas);

    // Particle settings - fewer particles on mobile
    const isMobile = window.innerWidth < 768;
    const particleCount = isMobile ? 30 : 80;
    const particles: Particle[] = [];

    // Initialize particles
    for (let i = 0; i < particleCount; i++) {
      particles.push({
        x: Math.random() * canvas.width,
        y: Math.random() * canvas.height,
        vx: (Math.random() - 0.5) * 0.5,
        vy: (Math.random() - 0.5) * 0.5,
        size: Math.random() * 2 + 1,
        opacity: Math.random() * 0.5 + 0.2,
      });
    }

    // Draw a single frame (positions are advanced by the caller when animating).
    const draw = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      particles.forEach((particle, i) => {
        ctx.beginPath();
        ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(255, 255, 255, ${particle.opacity})`;
        ctx.fill();

        // Draw connections
        particles.slice(i + 1).forEach(otherParticle => {
          const dx = particle.x - otherParticle.x;
          const dy = particle.y - otherParticle.y;
          const distance = Math.sqrt(dx * dx + dy * dy);

          if (distance < 120) {
            ctx.beginPath();
            ctx.moveTo(particle.x, particle.y);
            ctx.lineTo(otherParticle.x, otherParticle.y);
            const opacity = (1 - distance / 120) * 0.15;
            ctx.strokeStyle = `rgba(255, 255, 255, ${opacity})`;
            ctx.lineWidth = 0.5;
            ctx.stroke();
          }
        });
      });
    };

    // Respect reduced-motion: render a single static frame and skip the loop.
    const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reduceMotion) {
      draw();
      return () => window.removeEventListener('resize', resizeCanvas);
    }

    // Animation loop
    let animationFrameId = 0;
    const animate = () => {
      particles.forEach(particle => {
        particle.x += particle.vx;
        particle.y += particle.vy;
        if (particle.x < 0) particle.x = canvas.width;
        if (particle.x > canvas.width) particle.x = 0;
        if (particle.y < 0) particle.y = canvas.height;
        if (particle.y > canvas.height) particle.y = 0;
      });
      draw();
      animationFrameId = requestAnimationFrame(animate);
    };

    // Pause the rAF loop while the hero is scrolled off-screen to save CPU/battery.
    let running = false;
    const start = () => {
      if (!running) {
        running = true;
        animationFrameId = requestAnimationFrame(animate);
      }
    };
    const stop = () => {
      running = false;
      cancelAnimationFrame(animationFrameId);
    };

    const observer = new IntersectionObserver(
      entries => {
        if (entries[0].isIntersecting) {
          start();
        } else {
          stop();
        }
      },
      {threshold: 0},
    );
    observer.observe(canvas);
    start();

    return () => {
      window.removeEventListener('resize', resizeCanvas);
      observer.disconnect();
      stop();
    };
  }, []);

  return <canvas ref={canvasRef} className={styles.particleCanvas} />;
}
