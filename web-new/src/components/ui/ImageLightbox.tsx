/**
 * ImageLightbox Component
 * 
 * Full-screen modal for viewing images with zoom/pan support
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { X, Download, ZoomIn, ZoomOut, RotateCw } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';

interface ImageLightboxProps {
    src: string;
    fileName: string;
    onClose: () => void;
    onDownload: () => void;
}

export function ImageLightbox({ src, fileName, onClose, onDownload }: ImageLightboxProps) {
    const [scale, setScale] = useState(1);
    const [rotation, setRotation] = useState(0);
    const [isLoaded, setIsLoaded] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

    // Handle keyboard shortcuts
    useEffect(() => {
        function handleKeyDown(e: KeyboardEvent) {
            switch (e.key) {
                case 'Escape':
                    onClose();
                    break;
                case '+':
                case '=':
                    setScale(s => Math.min(s + 0.5, 5));
                    break;
                case '-':
                    setScale(s => Math.max(s - 0.5, 0.5));
                    break;
                case 'r':
                    setRotation(r => (r + 90) % 360);
                    break;
                case 's':
                    onDownload();
                    break;
            }
        }

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [onClose, onDownload]);

    // Handle backdrop click
    const handleBackdropClick = useCallback((e: React.MouseEvent) => {
        if (e.target === containerRef.current) {
            onClose();
        }
    }, [onClose]);

    // Zoom controls
    const zoomIn = () => setScale(s => Math.min(s + 0.5, 5));
    const zoomOut = () => setScale(s => Math.max(s - 0.5, 0.5));
    const rotate = () => setRotation(r => (r + 90) % 360);
    const resetZoom = () => {
        setScale(1);
        setRotation(0);
    };

    return (
        <div
            ref={containerRef}
            onClick={handleBackdropClick}
            className={cn(
                'fixed inset-0 z-50 flex items-center justify-center',
                'bg-black/90 backdrop-blur-sm',
                'animate-in fade-in-0 duration-200'
            )}
        >
            {/* Controls */}
            <div className="absolute top-4 right-4 flex items-center gap-2 z-10">
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={zoomOut}
                    className="text-white/80 hover:text-white hover:bg-white/10 rounded-full"
                    title="Zoom out (-)"
                >
                    <ZoomOut className="h-5 w-5" />
                </Button>
                <span className="text-white/70 text-sm min-w-[50px] text-center">
                    {Math.round(scale * 100)}%
                </span>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={zoomIn}
                    className="text-white/80 hover:text-white hover:bg-white/10 rounded-full"
                    title="Zoom in (+)"
                >
                    <ZoomIn className="h-5 w-5" />
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={rotate}
                    className="text-white/80 hover:text-white hover:bg-white/10 rounded-full"
                    title="Rotate (R)"
                >
                    <RotateCw className="h-5 w-5" />
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={onDownload}
                    className="text-white/80 hover:text-white hover:bg-white/10 rounded-full"
                    title="Download (S)"
                >
                    <Download className="h-5 w-5" />
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={onClose}
                    className="text-white/80 hover:text-white hover:bg-white/10 rounded-full ml-2"
                    title="Close (Esc)"
                >
                    <X className="h-6 w-6" />
                </Button>
            </div>

            {/* File name */}
            <div className="absolute top-4 left-4 text-white/70 text-sm max-w-[50%] truncate z-10">
                {fileName}
            </div>

            {/* Image */}
            <div className="relative max-w-[90vw] max-h-[90vh] flex items-center justify-center">
                {!isLoaded && (
                    <div className="absolute inset-0 flex items-center justify-center">
                        <div className="w-10 h-10 rounded-full border-2 border-white border-t-transparent animate-spin" />
                    </div>
                )}
                <img
                    src={src}
                    alt={fileName}
                    onLoad={() => setIsLoaded(true)}
                    onDoubleClick={resetZoom}
                    style={{
                        transform: `scale(${scale}) rotate(${rotation}deg)`,
                        transition: 'transform 0.2s ease-out',
                    }}
                    className={cn(
                        'max-w-[90vw] max-h-[90vh] object-contain cursor-zoom-in select-none',
                        isLoaded ? 'opacity-100' : 'opacity-0'
                    )}
                    draggable={false}
                />
            </div>

            {/* Hint */}
            <div className="absolute bottom-4 left-1/2 -translate-x-1/2 text-white/50 text-xs z-10">
                Double-click to reset • Scroll to zoom • Press Esc to close
            </div>
        </div>
    );
}

export default ImageLightbox;
