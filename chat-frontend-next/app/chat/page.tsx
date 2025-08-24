"use client"

import type React from "react"
import { useState, useRef, useEffect } from "react"
import { useAuth } from '@/hooks/useAuth';
import { useRouter } from 'next/navigation';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  Search,
  Send,
  MoreVertical,
  Phone,
  Video,
  Paperclip,
  ImageIcon,
  Folder,
  Smile,
  Settings,
  User,
  FileText,
  Download,
  X,
} from "lucide-react"
import { cn } from "@/lib/utils"
import Link from "next/link"

interface Chat {
  id: string
  name: string
  lastMessage: string
  time: string
  unread?: number
  avatar: string
  online?: boolean
}

interface Attachment {
  id: string
  name: string
  size: string
  type: "image" | "file" | "video" | "folder"
  url?: string
}

interface Message {
  id: string
  content: string
  time: string
  sender: "me" | "other"
  attachments?: Attachment[]
}

const mockChats: Chat[] = [
  {
    id: "1",
    name: "Sarah Wilson",
    lastMessage: "Hey, how's the project going?",
    time: "2m",
    unread: 2,
    avatar: "SW",
    online: true,
  },
  { id: "2", name: "Design Team", lastMessage: "The mockups look great!", time: "15m", unread: 5, avatar: "DT" },
  { id: "3", name: "Alex Chen", lastMessage: "Thanks for the feedback", time: "1h", avatar: "AC", online: true },
  { id: "4", name: "Marketing", lastMessage: "Campaign launch is tomorrow", time: "2h", avatar: "MK" },
  { id: "5", name: "David Kim", lastMessage: "Let's schedule a call", time: "3h", avatar: "DK" },
  { id: "6", name: "Product Team", lastMessage: "New features are ready", time: "1d", avatar: "PT" },
]

const mockMessages: Message[] = [
  { id: "1", content: "Hey, how's the project going?", time: "2:30 PM", sender: "other" },
  { id: "2", content: "It's going well! Just finished the wireframes", time: "2:32 PM", sender: "me" },
  {
    id: "3",
    content: "That's great to hear. Can you share them?",
    time: "2:33 PM",
    sender: "other",
    attachments: [
      { id: "1", name: "wireframes.pdf", size: "2.4 MB", type: "file" },
      { id: "2", name: "mockup.png", size: "1.8 MB", type: "image", url: "/placeholder-kxkes.png" },
    ],
  },
  { id: "4", content: "Sure, I'll send them over in a few minutes", time: "2:35 PM", sender: "me" },
  { id: "5", content: "Perfect! Looking forward to reviewing them", time: "2:36 PM", sender: "other" },
]

export default function ChatPage() {
  const { isAuthenticated, loading, user, logout } = useAuth();
  const router = useRouter();
  const [selectedChat, setSelectedChat] = useState<Chat>(mockChats[0])
  const [message, setMessage] = useState("")
  const [messages, setMessages] = useState<Message[]>(mockMessages)
  const [showAttachments, setShowAttachments] = useState(false)
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])
  const fileInputRef = useRef<HTMLInputElement>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, loading, router]);

  if (loading) return <div className="h-screen bg-white flex items-center justify-center"><div className="text-black">Loading...</div></div>;
  if (!isAuthenticated) return null;

  const handleSendMessage = () => {
    if (!message.trim() && selectedFiles.length === 0) return

    const attachments: Attachment[] = selectedFiles.map((file, index) => ({
      id: `${Date.now()}-${index}`,
      name: file.name,
      size: formatFileSize(file.size),
      type: getFileType(file.type),
      url: file.type.startsWith("image/") ? URL.createObjectURL(file) : undefined,
    }))

    const newMessage: Message = {
      id: Date.now().toString(),
      content: message,
      time: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      sender: "me",
      attachments: attachments.length > 0 ? attachments : undefined,
    }

    setMessages([...messages, newMessage])
    setMessage("")
    setSelectedFiles([])
    setShowAttachments(false)
  }

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return "0 Bytes"
    const k = 1024
    const sizes = ["Bytes", "KB", "MB", "GB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Number.parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
  }

  const getFileType = (mimeType: string): "image" | "file" | "video" | "folder" => {
    if (mimeType.startsWith("image/")) return "image"
    if (mimeType.startsWith("video/")) return "video"
    return "file"
  }

  const handleFileSelect = (type: "file" | "image") => {
    const input = type === "image" ? imageInputRef.current : fileInputRef.current
    input?.click()
  }

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    setSelectedFiles((prev) => [...prev, ...files])
    setShowAttachments(true)
  }

  const removeFile = (index: number) => {
    setSelectedFiles((prev) => prev.filter((_, i) => i !== index))
  }

  const renderAttachment = (attachment: Attachment) => {
    const getIcon = () => {
      switch (attachment.type) {
        case "image":
          return <ImageIcon className="h-4 w-4" />
        case "video":
          return <Video className="h-4 w-4" />
        case "folder":
          return <Folder className="h-4 w-4" />
        default:
          return <FileText className="h-4 w-4" />
      }
    }

    return (
      <div key={attachment.id} className="border border-gray-200 rounded-lg p-3 bg-gray-50 max-w-xs">
        {attachment.type === "image" && attachment.url ? (
          <div className="mb-2">
            <img
              src={attachment.url || "/placeholder.svg"}
              alt={attachment.name}
              className="w-full h-32 object-cover rounded"
            />
          </div>
        ) : null}
        <div className="flex items-center gap-2">
          <div className="p-2 bg-white rounded border border-gray-200">{getIcon()}</div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-black truncate">{attachment.name}</p>
            <p className="text-xs text-gray-500">{attachment.size}</p>
          </div>
          <Button variant="ghost" size="sm" className="h-8 w-8 p-0 hover:bg-gray-200">
            <Download className="h-3 w-3" />
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="h-screen bg-white flex">
      {/* Hidden file inputs */}
      <input ref={fileInputRef} type="file" multiple className="hidden" onChange={handleFileChange} accept="*/*" />
      <input ref={imageInputRef} type="file" multiple className="hidden" onChange={handleFileChange} accept="image/*" />

      {/* Chat List Sidebar */}
      <div className="w-80 border-r border-gray-200 bg-white flex flex-col">
        {/* Header */}
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-3">
            <h1 className="text-xl font-semibold text-black">Messages</h1>
            <div className="flex items-center gap-2">
              <Link href="/profile">
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0 hover:bg-gray-100">
                  <User className="h-4 w-4 text-gray-600" />
                </Button>
              </Link>
              <Link href="/chat/settings">
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0 hover:bg-gray-100">
                  <Settings className="h-4 w-4 text-gray-600" />
                </Button>
              </Link>
            </div>
          </div>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
            <Input
              placeholder="Search conversations..."
              className="pl-10 bg-gray-50 border-gray-200 focus-visible:ring-1 focus-visible:ring-black text-black placeholder:text-gray-400"
            />
          </div>
        </div>

        {/* Chat List */}
        <ScrollArea className="flex-1">
          <div className="p-2">
            {mockChats.map((chat) => (
              <div
                key={chat.id}
                onClick={() => setSelectedChat(chat)}
                className={cn(
                  "flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-all duration-200 hover:bg-gray-50",
                  selectedChat.id === chat.id && "bg-gray-100",
                )}
              >
                <div className="relative">
                  <Link href={`/profile/${chat.id}`}>
                    <Avatar className="h-12 w-12 cursor-pointer hover:opacity-80 transition-opacity">
                      <AvatarFallback className="bg-gray-100 text-black font-medium border border-gray-200">
                        {chat.avatar}
                      </AvatarFallback>
                    </Avatar>
                  </Link>
                  {chat.online && (
                    <div className="absolute -bottom-0.5 -right-0.5 h-3 w-3 bg-black border-2 border-white rounded-full" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between mb-1">
                    <h3 className="font-medium text-black truncate">{chat.name}</h3>
                    <span className="text-xs text-gray-500">{chat.time}</span>
                  </div>
                  <p className="text-sm text-gray-600 truncate">{chat.lastMessage}</p>
                </div>
                {chat.unread && (
                  <div className="bg-black text-white text-xs rounded-full h-5 w-5 flex items-center justify-center font-medium">
                    {chat.unread}
                  </div>
                )}
              </div>
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* Chat Window */}
      <div className="flex-1 flex flex-col">
        {/* Chat Header */}
        <div className="p-4 border-b border-gray-200 bg-white flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="relative">
              <Link href={`/profile/${selectedChat.id}`}>
                <Avatar className="h-10 w-10 cursor-pointer hover:opacity-80 transition-opacity">
                  <AvatarFallback className="bg-gray-100 text-black font-medium border border-gray-200">
                    {selectedChat.avatar}
                  </AvatarFallback>
                </Avatar>
              </Link>
              {selectedChat.online && (
                <div className="absolute -bottom-0.5 -right-0.5 h-3 w-3 bg-black border-2 border-white rounded-full" />
              )}
            </div>
            <div>
              <h2 className="font-semibold text-black">{selectedChat.name}</h2>
              <p className="text-xs text-gray-500">{selectedChat.online ? "Active now" : "Last seen 1h ago"}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" className="h-9 w-9 p-0 hover:bg-gray-100">
              <Phone className="h-4 w-4 text-gray-600" />
            </Button>
            <Button variant="ghost" size="sm" className="h-9 w-9 p-0 hover:bg-gray-100">
              <Video className="h-4 w-4 text-gray-600" />
            </Button>
            <Button variant="ghost" size="sm" className="h-9 w-9 p-0 hover:bg-gray-100">
              <MoreVertical className="h-4 w-4 text-gray-600" />
            </Button>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-hidden">
          <ScrollArea className="h-full p-4 bg-gray-50">
            <div className="space-y-4">
              {messages.map((msg) => (
                <div key={msg.id} className={cn("flex", msg.sender === "me" ? "justify-end" : "justify-start")}>
                  <div className="max-w-xs lg:max-w-md space-y-2">
                    {msg.attachments && msg.attachments.length > 0 && (
                      <div className="space-y-2">{msg.attachments.map(renderAttachment)}</div>
                    )}

                    {msg.content && (
                      <div
                        className={cn(
                          "px-4 py-2 rounded-2xl transition-all duration-200",
                          msg.sender === "me" ? "bg-black text-white" : "bg-white text-black border border-gray-200",
                        )}
                      >
                        <p className="text-sm">{msg.content}</p>
                        <p className={cn("text-xs mt-1", msg.sender === "me" ? "text-gray-300" : "text-gray-500")}>
                          {msg.time}
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        </div>

        {/* Message Input */}
        <div className="p-4 border-t border-gray-200 bg-white">
          {(showAttachments || selectedFiles.length > 0) && (
            <div className="mb-3 p-3 bg-gray-50 rounded-lg border border-gray-200">
              {selectedFiles.length > 0 ? (
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-medium text-black">Selected Files ({selectedFiles.length})</h4>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setSelectedFiles([])
                        setShowAttachments(false)
                      }}
                      className="h-6 w-6 p-0 hover:bg-gray-200"
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                  <div className="space-y-2 max-h-32 overflow-y-auto">
                    {selectedFiles.map((file, index) => (
                      <div key={index} className="flex items-center gap-3 p-2 bg-white rounded border border-gray-200">
                        <div className="p-1 bg-gray-100 rounded">
                          {file.type.startsWith("image/") ? (
                            <ImageIcon className="h-4 w-4 text-green-600" />
                          ) : file.type.startsWith("video/") ? (
                            <Video className="h-4 w-4 text-purple-600" />
                          ) : (
                            <FileText className="h-4 w-4 text-blue-600" />
                          )}
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-black truncate">{file.name}</p>
                          <p className="text-xs text-gray-500">{formatFileSize(file.size)}</p>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => removeFile(index)}
                          className="h-6 w-6 p-0 hover:bg-gray-200"
                        >
                          <X className="h-3 w-3" />
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="grid grid-cols-4 gap-3">
                  <button
                    onClick={() => handleFileSelect("file")}
                    className="flex flex-col items-center gap-2 p-3 rounded-lg hover:bg-gray-100 transition-colors"
                  >
                    <div className="h-10 w-10 bg-blue-100 rounded-full flex items-center justify-center">
                      <Paperclip className="h-5 w-5 text-blue-600" />
                    </div>
                    <span className="text-xs text-gray-600">File</span>
                  </button>
                  <button
                    onClick={() => handleFileSelect("image")}
                    className="flex flex-col items-center gap-2 p-3 rounded-lg hover:bg-gray-100 transition-colors"
                  >
                    <div className="h-10 w-10 bg-green-100 rounded-full flex items-center justify-center">
                      <ImageIcon className="h-5 w-5 text-green-600" />
                    </div>
                    <span className="text-xs text-gray-600">Photo</span>
                  </button>
                  <button className="flex flex-col items-center gap-2 p-3 rounded-lg hover:bg-gray-100 transition-colors">
                    <div className="h-10 w-10 bg-orange-100 rounded-full flex items-center justify-center">
                      <Folder className="h-5 w-5 text-orange-600" />
                    </div>
                    <span className="text-xs text-gray-600">Folder</span>
                  </button>
                  <button className="flex flex-col items-center gap-2 p-3 rounded-lg hover:bg-gray-100 transition-colors">
                    <div className="h-10 w-10 bg-purple-100 rounded-full flex items-center justify-center">
                      <Video className="h-5 w-5 text-purple-600" />
                    </div>
                    <span className="text-xs text-gray-600">Video</span>
                  </button>
                </div>
              )}
            </div>
          )}

          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-10 w-10 p-0 hover:bg-gray-100"
              onClick={() => setShowAttachments(!showAttachments)}
            >
              <Paperclip className="h-4 w-4 text-gray-600" />
            </Button>

            <Input
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Type a message..."
              className="flex-1 bg-gray-50 border-gray-200 focus-visible:ring-1 focus-visible:ring-black text-black placeholder:text-gray-400"
              onKeyPress={(e) => e.key === "Enter" && handleSendMessage()}
            />

            <Button variant="ghost" size="sm" className="h-10 w-10 p-0 hover:bg-gray-100">
              <Smile className="h-4 w-4 text-gray-600" />
            </Button>

            <Button
              onClick={handleSendMessage}
              size="sm"
              className="h-10 w-10 p-0 bg-black hover:bg-gray-800 text-white transition-colors"
            >
              <Send className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
