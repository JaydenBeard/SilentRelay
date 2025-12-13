import { useEffect, useRef, useState } from 'react';

/**
 * Hook to detect when an element enters the viewport
 * Returns a ref to attach to the element and a boolean indicating if it's in view
 */
export function useInView(options?: IntersectionObserverInit) {
    const ref = useRef<HTMLDivElement>(null);
    const [inView, setInView] = useState(false);

    useEffect(() => {
        const element = ref.current;
        if (!element) return;

        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting) {
                    setInView(true);
                    // Once in view, stop observing (animation only plays once)
                    observer.unobserve(element);
                }
            },
            {
                threshold: 0.1,
                rootMargin: '0px 0px -50px 0px',
                ...options,
            }
        );

        observer.observe(element);

        return () => observer.disconnect();
    }, [options]);

    return { ref, inView };
}

/**
 * Hook to observe multiple elements for scroll animations
 * Call this once at the page level to animate all .animate-on-scroll elements
 */
export function useScrollAnimations() {
    useEffect(() => {
        const elements = document.querySelectorAll('.animate-on-scroll');

        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        entry.target.classList.add('in-view');
                        observer.unobserve(entry.target);
                    }
                });
            },
            {
                threshold: 0.1,
                rootMargin: '0px 0px -50px 0px',
            }
        );

        elements.forEach((el) => observer.observe(el));

        return () => observer.disconnect();
    }, []);
}
