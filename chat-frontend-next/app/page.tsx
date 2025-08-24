"use client";
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import api from '@/lib/api';
import { User } from '@/lib/types';
import {
  AccountCircle,
  CheckCircle,
  Chat,
  Group,
  Add,
  GroupAdd,
  Settings,
  HelpOutline,
  ExitToApp,
  Dashboard
} from '@mui/icons-material';

export default function Home() {
  const { isAuthenticated, loading, logout } = useAuth();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [loadingProfile, setLoadingProfile] = useState(true);

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      router.push('/login');
    }
    
    if (isAuthenticated) {
      const fetchProfile = async () => {
        try {
          const response = await api.get('/auth/profile');
          setUser(response.data);
        } catch (error) {
          console.error('Failed to fetch profile:', error);
        } finally {
          setLoadingProfile(false);
        }
      };
      fetchProfile();
    }
  }, [isAuthenticated, loading, router]);

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
  };

  const GRADIENTS = {
    primary: 'from-blue-600 to-purple-600',
    primaryHover: 'from-blue-700 to-purple-700',
    background: 'from-slate-50 via-blue-50 to-indigo-50'
  };

  if (loading) return <div className="h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100"><div className="flex flex-col items-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-4"></div><div className="text-gray-600">Loading...</div></div></div>;
  if (!isAuthenticated) return null;

  const handleLogout = () => {
    logout();
    router.push('/login');
  };

  return (
    <div className={`min-h-screen bg-gradient-to-br ${GRADIENTS.background}`}>
      {/* Modern Header */}
      <header className="bg-white/80 backdrop-blur-md border-b border-gray-200/50 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-3">
              <div className="w-8 h-8 bg-gradient-to-r from-blue-600 to-purple-600 rounded-lg flex items-center justify-center">
                <Chat className="w-5 h-5 text-white" />
              </div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-gray-900 to-gray-600 bg-clip-text text-transparent">
                WindGo Chat
              </h1>
            </div>
            <div className="flex items-center space-x-4">
              {user && (
                <div className="flex items-center space-x-3 bg-gray-50/50 rounded-full px-4 py-2">
                  <div className="text-right">
                    <div className="text-sm font-medium text-gray-900">{user.username}</div>
                    <div className="text-xs text-gray-500">{user.email}</div>
                  </div>
                  <div className="w-8 h-8 bg-gradient-to-r from-blue-500 to-purple-600 rounded-full flex items-center justify-center">
                    <span className="text-white text-sm font-semibold">
                      {user.username.charAt(0).toUpperCase()}
                    </span>
                  </div>
                </div>
              )}
              <Button 
                onClick={handleLogout} 
                variant="outline" 
                className="border-gray-300 hover:bg-gray-50"
              >
                <ExitToApp className="w-4 h-4 mr-2" />
                Sign Out
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {loadingProfile ? (
          <div className="flex flex-col items-center justify-center py-20">
            <div className="w-12 h-12 border-4 border-blue-200 border-t-blue-600 rounded-full animate-spin mb-4"></div>
            <p className="text-gray-600 font-medium">Loading your dashboard...</p>
          </div>
        ) : (
          <div className="space-y-8">
            {/* Hero Welcome Section */}
            <div className="relative overflow-hidden bg-white rounded-2xl shadow-sm border border-gray-100">
              <div className="absolute inset-0 bg-gradient-to-r from-blue-600/5 to-purple-600/5"></div>
              <div className="relative px-8 py-12 text-center">
                <div className="w-16 h-16 bg-gradient-to-r from-blue-500 to-purple-600 rounded-2xl flex items-center justify-center mx-auto mb-6">
                  <Dashboard className="w-8 h-8 text-white" />
                </div>
                <h2 className="text-3xl font-bold text-gray-900 mb-4">
                  Welcome back, {user?.username}! 👋
                </h2>
                <p className="text-lg text-gray-600 max-w-2xl mx-auto leading-relaxed mb-8">
                  Your secure messaging platform is ready. Connect with friends, join conversations, and stay in touch with the world.
                </p>
                <div className="flex justify-center space-x-4">
                  <Button
                    onClick={() => router.push('/chat')}
                    className={`bg-gradient-to-r ${GRADIENTS.primary} hover:${GRADIENTS.primaryHover} text-white px-8 py-3 rounded-xl font-semibold shadow-lg hover:shadow-xl transition-all duration-200`}
                  >
                    <Chat className="w-5 h-5 mr-2" />
                    Go to Chat
                  </Button>
                </div>
              </div>
            </div>

            {/* Modern Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 hover:shadow-md transition-shadow">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600 mb-1">Account Status</p>
                    <p className="text-2xl font-bold text-green-600">Active</p>
                  </div>
                  <div className="w-12 h-12 bg-green-100 rounded-xl flex items-center justify-center">
                    <CheckCircle className="w-6 h-6 text-green-600" />
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 hover:shadow-md transition-shadow">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600 mb-1">Messages</p>
                    <p className="text-2xl font-bold text-blue-600">0</p>
                  </div>
                  <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center">
                    <Chat className="w-6 h-6 text-blue-600" />
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 hover:shadow-md transition-shadow">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600 mb-1">Chat Rooms</p>
                    <p className="text-2xl font-bold text-purple-600">0</p>
                  </div>
                  <div className="w-12 h-12 bg-purple-100 rounded-xl flex items-center justify-center">
                    <Group className="w-6 h-6 text-purple-600" />
                  </div>
                </div>
              </div>
            </div>

            {/* Modern Quick Actions */}
            <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-8">
              <div className="mb-8">
                <h3 className="text-xl font-bold text-gray-900 mb-2">Quick Actions</h3>
                <p className="text-gray-600">Start chatting, join rooms, or manage your account</p>
              </div>
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                <button className="group flex flex-col items-center justify-center p-6 rounded-xl border-2 border-gray-200 hover:border-blue-300 hover:bg-blue-50/50 transition-all duration-200">
                  <div className="w-12 h-12 bg-blue-100 group-hover:bg-blue-200 rounded-xl flex items-center justify-center mb-3 transition-colors">
                    <Add className="w-6 h-6 text-blue-600" />
                  </div>
                  <span className="font-medium text-gray-900 group-hover:text-blue-700">New Chat</span>
                  <span className="text-xs text-gray-500 mt-1">Start a conversation</span>
                </button>
                
                <button className="group flex flex-col items-center justify-center p-6 rounded-xl border-2 border-gray-200 hover:border-green-300 hover:bg-green-50/50 transition-all duration-200">
                  <div className="w-12 h-12 bg-green-100 group-hover:bg-green-200 rounded-xl flex items-center justify-center mb-3 transition-colors">
                    <GroupAdd className="w-6 h-6 text-green-600" />
                  </div>
                  <span className="font-medium text-gray-900 group-hover:text-green-700">Join Room</span>
                  <span className="text-xs text-gray-500 mt-1">Connect with groups</span>
                </button>
                
                <button className="group flex flex-col items-center justify-center p-6 rounded-xl border-2 border-gray-200 hover:border-orange-300 hover:bg-orange-50/50 transition-all duration-200">
                  <div className="w-12 h-12 bg-orange-100 group-hover:bg-orange-200 rounded-xl flex items-center justify-center mb-3 transition-colors">
                    <Settings className="w-6 h-6 text-orange-600" />
                  </div>
                  <span className="font-medium text-gray-900 group-hover:text-orange-700">Settings</span>
                  <span className="text-xs text-gray-500 mt-1">Manage preferences</span>
                </button>
                
                <button className="group flex flex-col items-center justify-center p-6 rounded-xl border-2 border-gray-200 hover:border-purple-300 hover:bg-purple-50/50 transition-all duration-200">
                  <div className="w-12 h-12 bg-purple-100 group-hover:bg-purple-200 rounded-xl flex items-center justify-center mb-3 transition-colors">
                    <HelpOutline className="w-6 h-6 text-purple-600" />
                  </div>
                  <span className="font-medium text-gray-900 group-hover:text-purple-700">Help</span>
                  <span className="text-xs text-gray-500 mt-1">Get support</span>
                </button>
              </div>
            </div>

            {/* Modern Account Info */}
            <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-8">
              <div className="flex items-center mb-8">
                <div className="w-12 h-12 bg-gray-100 rounded-xl flex items-center justify-center mr-4">
                  <AccountCircle className="w-6 h-6 text-gray-600" />
                </div>
                <div>
                  <h3 className="text-xl font-bold text-gray-900">Account Information</h3>
                  <p className="text-gray-600">Your profile details and account settings</p>
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                <div className="space-y-1">
                  <label className="text-sm font-medium text-gray-500 uppercase tracking-wide">Username</label>
                  <p className="text-lg font-semibold text-gray-900">{user?.username}</p>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium text-gray-500 uppercase tracking-wide">Email Address</label>
                  <p className="text-lg font-semibold text-gray-900">{user?.email}</p>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium text-gray-500 uppercase tracking-wide">Member Since</label>
                  <p className="text-lg font-semibold text-gray-900">{user?.created_at ? formatDate(user.created_at) : 'N/A'}</p>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium text-gray-500 uppercase tracking-wide">Last Updated</label>
                  <p className="text-lg font-semibold text-gray-900">{user?.updated_at ? formatDate(user.updated_at) : 'N/A'}</p>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
