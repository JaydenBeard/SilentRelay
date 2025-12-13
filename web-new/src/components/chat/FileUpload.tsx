/**
 * File Upload Component
 *
 * Handles file selection, preview, and upload with progress.
 */

import { useState, useRef, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Paperclip, X, FileText, Image, Video, Music, Upload, Loader2 } from 'lucide-react';
import { uploadFile, type UploadProgress, type UploadResult } from '@/core/services/fileUpload';
import { validateFile, getMediaType } from '@/core/crypto/media';
import { cn } from '@/lib/utils';

interface FileUploadProps {
  onUploadComplete: (result: UploadResult) => void;
  onError: (error: string) => void;
}

export function FileUpload({ onUploadComplete, onError }: FileUploadProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const validation = validateFile(file);
    if (!validation.valid) {
      onError(validation.error || 'Invalid file');
      return;
    }

    setSelectedFile(file);

    // Generate preview for images
    if (file.type.startsWith('image/')) {
      const reader = new FileReader();
      reader.onload = (e) => setPreview(e.target?.result as string);
      reader.readAsDataURL(file);
    } else {
      setPreview(null);
    }

    setIsOpen(true);
  }, [onError]);

  const handleUpload = async () => {
    if (!selectedFile) return;

    setIsUploading(true);
    setProgress(null);

    try {
      const result = await uploadFile(selectedFile, setProgress);
      onUploadComplete(result);
      handleClose();
    } catch (error) {
      onError(error instanceof Error ? error.message : 'Upload failed');
    } finally {
      setIsUploading(false);
    }
  };

  const handleClose = () => {
    setIsOpen(false);
    setSelectedFile(null);
    setPreview(null);
    setProgress(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const getFileIcon = (mimeType: string) => {
    const mediaType = getMediaType(mimeType);
    switch (mediaType) {
      case 'image':
        return <Image className="h-8 w-8" />;
      case 'video':
        return <Video className="h-8 w-8" />;
      case 'audio':
        return <Music className="h-8 w-8" />;
      default:
        return <FileText className="h-8 w-8" />;
    }
  };

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileSelect}
        accept="image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.txt"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={() => fileInputRef.current?.click()}
      >
        <Paperclip className="h-5 w-5" />
      </Button>

      <Dialog open={isOpen} onOpenChange={(open) => !isUploading && !open && handleClose()}>
        <DialogContent className="sm:max-w-md" aria-describedby="file-upload-description">
          <DialogHeader>
            <DialogTitle>Send File</DialogTitle>
            <DialogDescription id="file-upload-description">
              Preview and upload your file to send in the conversation
            </DialogDescription>
          </DialogHeader>

          {selectedFile && (
            <div className="space-y-4">
              {/* Preview */}
              <div className="relative rounded-lg overflow-hidden bg-background-tertiary">
                {preview ? (
                  <img
                    src={preview}
                    alt="Preview"
                    className="w-full h-48 object-contain"
                  />
                ) : (
                  <div className="w-full h-48 flex flex-col items-center justify-center text-foreground-muted">
                    {getFileIcon(selectedFile.type)}
                    <span className="mt-2 text-sm">{getMediaType(selectedFile.type)}</span>
                  </div>
                )}
                {!isUploading && (
                  <button
                    onClick={handleClose}
                    className="absolute top-2 right-2 p-1 rounded-full bg-background/80 hover:bg-background transition-colors"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>

              {/* File info */}
              <div className="space-y-1">
                <p className="text-sm font-medium truncate">{selectedFile.name}</p>
                <p className="text-xs text-foreground-muted">
                  {formatFileSize(selectedFile.size)}
                </p>
              </div>

              {/* Progress bar */}
              {progress && (
                <div className="space-y-1">
                  <div className="flex justify-between text-xs text-foreground-muted">
                    <span>Uploading...</span>
                    <span>{progress.percentage}%</span>
                  </div>
                  <div className="h-2 bg-background-tertiary rounded-full overflow-hidden">
                    <div
                      className="h-full bg-primary transition-all duration-300"
                      style={{ width: `${progress.percentage}%` }}
                    />
                  </div>
                </div>
              )}

              {/* Actions */}
              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={handleClose}
                  disabled={isUploading}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleUpload}
                  disabled={isUploading}
                >
                  {isUploading ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Uploading...
                    </>
                  ) : (
                    <>
                      <Upload className="h-4 w-4 mr-2" />
                      Send
                    </>
                  )}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

/**
 * File Message Display Component
 */
interface FileMessageProps {
  metadata: {
    type: string;
    mediaType: string;
    fileName: string;
    fileSize: number;
    mimeType: string;
    mediaId: string;
    thumbnail?: string;
  };
  isOwn: boolean;
  onDownload: () => void;
  isDownloading?: boolean;
}

export function FileMessage({ metadata, isOwn, onDownload, isDownloading }: FileMessageProps) {
  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const getFileIcon = () => {
    switch (metadata.mediaType) {
      case 'image':
        return <Image className="h-5 w-5" />;
      case 'video':
        return <Video className="h-5 w-5" />;
      case 'audio':
        return <Music className="h-5 w-5" />;
      default:
        return <FileText className="h-5 w-5" />;
    }
  };

  return (
    <div
      className={cn(
        'rounded-lg overflow-hidden max-w-xs cursor-pointer hover:opacity-90 transition-opacity',
        isOwn ? 'bg-message-outgoing' : 'bg-message-incoming'
      )}
      onClick={onDownload}
    >
      {/* Thumbnail for images */}
      {metadata.thumbnail && metadata.mediaType === 'image' && (
        <img
          src={metadata.thumbnail}
          alt={metadata.fileName}
          className="w-full h-32 object-cover"
        />
      )}

      {/* File info */}
      <div className="p-3 flex items-center gap-3">
        <div className={cn(
          'p-2 rounded-lg',
          isOwn ? 'bg-white/10' : 'bg-background-tertiary'
        )}>
          {isDownloading ? (
            <Loader2 className="h-5 w-5 animate-spin" />
          ) : (
            getFileIcon()
          )}
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium truncate">{metadata.fileName}</p>
          <p className={cn(
            'text-xs',
            isOwn ? 'text-white/70' : 'text-foreground-muted'
          )}>
            {formatFileSize(metadata.fileSize)}
          </p>
        </div>
      </div>
    </div>
  );
}
