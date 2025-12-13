/**
 * MessageRequests Component
 *
 * Shows pending message requests with Accept/Decline/Block options.
 */

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Check, X, Ban, MessageSquare, Shield } from 'lucide-react';
import type { Conversation } from '@/core/types';

interface MessageRequestsProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  requests: Conversation[];
  onAccept: (conversationId: string) => void;
  onDecline: (conversationId: string) => void;
  onBlock: (conversationId: string) => void;
  onSelectConversation: (conversationId: string) => void;
}

export function MessageRequests({
  open,
  onOpenChange,
  requests,
  onAccept,
  onDecline,
  onBlock,
  onSelectConversation,
}: MessageRequestsProps) {
  const handleAccept = (id: string) => {
    onAccept(id);
    onSelectConversation(id);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md max-h-[80vh]" aria-describedby="message-requests-description">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5" />
            Message Requests
          </DialogTitle>
          <DialogDescription id="message-requests-description">
            Messages from people you haven't chatted with before. They won't know you've seen their request until you accept.
          </DialogDescription>
        </DialogHeader>

        {requests.length === 0 ? (
          <div className="py-8 text-center">
            <div className="w-16 h-16 rounded-full bg-background-tertiary flex items-center justify-center mx-auto mb-4">
              <Shield className="h-8 w-8 text-foreground-muted" />
            </div>
            <p className="text-foreground-secondary font-medium">No message requests</p>
            <p className="text-sm text-foreground-muted mt-1">
              When someone new messages you, it will appear here
            </p>
          </div>
        ) : (
          <ScrollArea className="max-h-[50vh]">
            <div className="space-y-3">
              {requests.map((request) => (
                <div
                  key={request.id}
                  className="p-4 rounded-lg bg-background-tertiary/50 border border-border"
                >
                  <div className="flex items-start gap-3">
                    <Avatar className="h-12 w-12 flex-shrink-0">
                      <AvatarImage src={request.recipientAvatar} />
                      <AvatarFallback className="bg-background-tertiary">
                        {request.recipientName.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium truncate">{request.recipientName}</p>
                      {request.lastMessage && (
                        <p className="text-sm text-foreground-muted mt-1 line-clamp-2">
                          {request.lastMessage.content}
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 mt-4">
                    <Button
                      size="sm"
                      className="flex-1"
                      onClick={() => handleAccept(request.id)}
                    >
                      <Check className="h-4 w-4 mr-1" />
                      Accept
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1"
                      onClick={() => onDecline(request.id)}
                    >
                      <X className="h-4 w-4 mr-1" />
                      Decline
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => onBlock(request.id)}
                    >
                      <Ban className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </DialogContent>
    </Dialog>
  );
}

