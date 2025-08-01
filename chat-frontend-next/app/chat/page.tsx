"use client";
import { useAuth } from '@/hooks/useAuth';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { 
  Search, 
  Send, 
  MoreVert, 
  EmojiEmotions, 
  AttachFile,
  Menu,
  Settings,
  ExitToApp,
  Circle
} from '@mui/icons-material';

interface Message {
  id: number;
  user: string;
  content: string;
  time: string;
  isOwn: boolean;
}

interface ChatRoom {
  id: number;
  name: string;
  lastMessage: string;
  time: string;
  unread: number;
  online: boolean;
}

export default function ChatPage() {
  const { isAuthenticated, loading, user, logout } = useAuth();
  const router = useRouter();
  const [selectedRoom, setSelectedRoom] = useState<number>(1);
  const [message, setMessage] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  // Mock data
  const [rooms] = useState<ChatRoom[]>([
    { id: 1, name: 'General', lastMessage: 'Welcome to WindGo Chat!', time: '10:30', unread: 0, online: true },
    { id: 2, name: 'Random', lastMessage: 'Anyone here?', time: '09:45', unread: 2, online: true },
    { id: 3, name: 'Tech Talk', lastMessage: 'Check out this new framework', time: 'Yesterday', unread: 0, online: false },
    { id: 4, name: 'Design', lastMessage: 'What do you think about this UI?', time: 'Yesterday', unread: 1, online: true },
  ]);

  const [messages] = useState<Message[]>([
    { id: 1, user: 'Admin', content: 'Welcome to WindGo Chat! 🎉', time: '10:30', isOwn: false },
    { id: 2, user: user?.username || 'You', content: 'Thanks! Happy to be here.', time: '10:31', isOwn: true },
    { id: 3, user: 'Demo User', content: 'This looks great! Love the design.', time: '10:32', isOwn: false },
    { id: 4, user: user?.username || 'You', content: 'Agreed! Very clean and modern.', time: '10:33', isOwn: true },
  ]);

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, loading, router]);

  const handleSendMessage = () => {
    if (message.trim()) {
      // TODO: Implement message sending
      console.log('Sending message:', message);
      setMessage('');
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto"></div>
          <p className="mt-2 text-gray-600 text-sm">Loading...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) return null;

  return (
    <div className="h-screen bg-gray-50 flex">
      {/* Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
        {/* Header */}
        <div className="p-4 border-b border-gray-100">
          <div className="flex items-center justify-between mb-3">
            <h1 className="text-xl font-semibold text-gray-800">WindGo</h1>
            <div className="flex items-center space-x-2">
              <button className="p-2 hover:bg-gray-100 rounded-full transition-colors">
                <Settings className="w-5 h-5 text-gray-600" />
              </button>
              <button 
                onClick={logout}
                className="p-2 hover:bg-gray-100 rounded-full transition-colors"
              >
                <ExitToApp className="w-5 h-5 text-gray-600" />
              </button>
            </div>
          </div>
          
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search chats..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
            />
          </div>
        </div>

        {/* Chat List */}
        <div className="flex-1 overflow-y-auto">
          {rooms.map((room) => (
            <div
              key={room.id}
              onClick={() => setSelectedRoom(room.id)}
              className={`p-4 border-b border-gray-50 cursor-pointer hover:bg-gray-50 transition-colors ${
                selectedRoom === room.id ? 'bg-blue-50 border-r-2 border-r-blue-500' : ''
              }`}
            >
              <div className="flex items-center space-x-3">
                <div className="relative">
                  <div className="w-12 h-12 bg-gradient-to-br from-blue-400 to-blue-600 rounded-full flex items-center justify-center text-white font-semibold">
                    {room.name[0]}
                  </div>
                  {room.online && (
                    <Circle className="absolute bottom-0 right-0 w-3 h-3 text-green-400 bg-white rounded-full" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <h3 className="font-medium text-gray-900 truncate">{room.name}</h3>
                    <span className="text-xs text-gray-500">{room.time}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <p className="text-sm text-gray-600 truncate">{room.lastMessage}</p>
                    {room.unread > 0 && (
                      <span className="bg-blue-500 text-white text-xs rounded-full px-2 py-0.5 min-w-[18px] text-center">
                        {room.unread}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* User Info */}
        <div className="p-4 border-t border-gray-100 bg-gray-50">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-gradient-to-br from-green-400 to-green-600 rounded-full flex items-center justify-center text-white font-semibold">
              {user?.username?.[0]?.toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-medium text-gray-900 truncate">{user?.username}</p>
              <p className="text-sm text-gray-600 truncate">{user?.role}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col bg-white">
        {/* Chat Header */}
        <div className="px-6 py-4 border-b border-gray-100 bg-white">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-400 to-blue-600 rounded-full flex items-center justify-center text-white font-semibold">
                {rooms.find(r => r.id === selectedRoom)?.name[0]}
              </div>
              <div>
                <h2 className="font-semibold text-gray-900">
                  {rooms.find(r => r.id === selectedRoom)?.name}
                </h2>
                <p className="text-sm text-gray-500">5 members, 3 online</p>
              </div>
            </div>
            <button className="p-2 hover:bg-gray-100 rounded-full transition-colors">
              <MoreVert className="w-5 h-5 text-gray-600" />
            </button>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4 bg-gray-50">
          {messages.map((msg) => (
            <div key={msg.id} className={`flex ${msg.isOwn ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-xs lg:max-w-md ${msg.isOwn ? 'order-2' : 'order-1'}`}>
                <div className={`px-4 py-2 rounded-2xl ${
                  msg.isOwn 
                    ? 'bg-blue-500 text-white rounded-br-md' 
                    : 'bg-white text-gray-900 rounded-bl-md shadow-sm'
                }`}>
                  {!msg.isOwn && (
                    <p className="text-xs font-medium text-blue-600 mb-1">{msg.user}</p>
                  )}
                  <p className="text-sm">{msg.content}</p>
                </div>
                <p className={`text-xs text-gray-500 mt-1 ${msg.isOwn ? 'text-right' : 'text-left'}`}>
                  {msg.time}
                </p>
              </div>
            </div>
          ))}
        </div>

        {/* Message Input */}
        <div className="p-4 border-t border-gray-100 bg-white">
          <div className="flex items-end space-x-3">
            <button className="p-2 hover:bg-gray-100 rounded-full transition-colors">
              <AttachFile className="w-5 h-5 text-gray-600" />
            </button>
            <div className="flex-1 relative">
              <input
                type="text"
                placeholder="Type a message..."
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
                className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-2xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
              />
              <button className="absolute right-3 top-1/2 transform -translate-y-1/2 p-1 hover:bg-gray-200 rounded-full transition-colors">
                <EmojiEmotions className="w-5 h-5 text-gray-600" />
              </button>
            </div>
            <button
              onClick={handleSendMessage}
              disabled={!message.trim()}
              className="p-3 bg-blue-500 hover:bg-blue-600 disabled:bg-gray-300 text-white rounded-full transition-colors"
            >
              <Send className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
