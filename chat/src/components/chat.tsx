"use client";

import {useMemo, useState} from "react";
import {ListTree} from "lucide-react";
import {useChat} from "./chat-provider";
import MessageInput from "./message-input";
import MessageList from "./message-list";
import {TimelinePanel} from "./timeline-panel";
import {Button} from "@/components/ui/button";
import {groupToolsByTurn} from "@/lib/timeline";

export function Chat() {
  const {messages, timeline, loading, sendMessage, serverStatus} = useChat();
  const [panelOpen, setPanelOpen] = useState(false);
  const toolTurns = useMemo(() => groupToolsByTurn(timeline), [timeline]);

  return (
    <div className="relative flex flex-1 min-h-0">
      <div className="flex flex-col flex-1 min-w-0">
        <MessageList messages={messages} toolTurns={toolTurns}/>
        <MessageInput
          onSendMessage={sendMessage}
          disabled={loading}
          serverStatus={serverStatus}
        />
      </div>
      {!panelOpen && timeline.length > 0 && (
        <Button
          variant="outline"
          size="icon"
          className="absolute top-2 right-2 z-10 size-8"
          onClick={() => setPanelOpen(true)}
          aria-label="Open timeline"
          title="Timeline"
        >
          <ListTree className="size-4" />
        </Button>
      )}
      {panelOpen && (
        <TimelinePanel timeline={timeline} onClose={() => setPanelOpen(false)} />
      )}
    </div>
  );
}
