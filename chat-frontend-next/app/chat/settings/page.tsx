"use client"

import { useState, useEffect } from "react"
import { useAuth } from '@/hooks/useAuth';
import { useRouter } from 'next/navigation';
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { ArrowLeft, Bell, Moon, Shield, Download, Trash2, LogOut, Camera } from "lucide-react"
import Link from "next/link"

export default function ChatSettingsPage() {
    const { isAuthenticated, loading, user, logout } = useAuth();
    const router = useRouter();
    const [settings, setSettings] = useState({
        notifications: true,
        soundEnabled: true,
        darkMode: false,
        readReceipts: true,
        onlineStatus: true,
        messagePreview: true,
        autoDownload: false,
        twoFactorAuth: false,
    })

    useEffect(() => {
        if (!loading && !isAuthenticated) {
            router.push('/login');
        }
    }, [isAuthenticated, loading, router]);

    if (loading) return <div className="h-screen bg-white flex items-center justify-center"><div className="text-black">Loading...</div></div>;
    if (!isAuthenticated) return null;

    const updateSetting = (key: string, value: boolean) => {
        setSettings((prev) => ({ ...prev, [key]: value }))
    }

    const getInitials = (name: string) => name?.split(' ').map(n => n[0]).join('').toUpperCase() || 'U';

    return (
        <div className="min-h-screen bg-white">
            {/* Header */}
            <div className="border-b border-gray-200 bg-white">
                <div className="max-w-4xl mx-auto px-4 py-4">
                    <div className="flex items-center gap-4">
                        <Link href="/chat">
                            <Button variant="ghost" size="sm" className="h-9 w-9 p-0 hover:bg-gray-100">
                                <ArrowLeft className="h-4 w-4 text-gray-600" />
                            </Button>
                        </Link>
                        <h1 className="text-xl font-semibold text-black">Settings</h1>
                    </div>
                </div>
            </div>

            {/* Settings Content */}
            <div className="max-w-4xl mx-auto px-4 py-6">
                <div className="space-y-6">
                    {/* Profile Section */}
                    <div className="bg-white border border-gray-200 rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-black mb-4">Profile</h2>
                        <div className="flex items-center gap-4">
                            <div className="relative">
                                <Avatar className="h-16 w-16 border-2 border-gray-200">
                                    <AvatarFallback className="bg-gray-100 text-black text-xl font-medium">
                                        {getInitials(user?.username || 'User')}
                                    </AvatarFallback>
                                </Avatar>
                                <Button
                                    size="sm"
                                    className="absolute -bottom-1 -right-1 h-7 w-7 p-0 bg-black hover:bg-gray-800 text-white rounded-full"
                                >
                                    <Camera className="h-3 w-3" />
                                </Button>
                            </div>
                            <div className="flex-1">
                                <h3 className="font-medium text-black">{user?.username || 'John Doe'}</h3>
                                <p className="text-sm text-gray-600">{user?.email || 'john.doe@example.com'}</p>
                                <Link href="/profile">
                                    <Button variant="ghost" size="sm" className="mt-2 h-8 px-3 hover:bg-gray-100">
                                        Edit Profile
                                    </Button>
                                </Link>
                            </div>
                        </div>
                    </div>

                    {/* Notifications */}
                    <div className="bg-white border border-gray-200 rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-black mb-4 flex items-center gap-2">
                            <Bell className="h-5 w-5" />
                            Notifications
                        </h2>
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Push Notifications</h3>
                                    <p className="text-sm text-gray-600">Receive notifications for new messages</p>
                                </div>
                                <Switch
                                    checked={settings.notifications}
                                    onCheckedChange={(checked) => updateSetting("notifications", checked)}
                                />
                            </div>
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Sound</h3>
                                    <p className="text-sm text-gray-600">Play sound for new messages</p>
                                </div>
                                <Switch
                                    checked={settings.soundEnabled}
                                    onCheckedChange={(checked) => updateSetting("soundEnabled", checked)}
                                />
                            </div>
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Message Preview</h3>
                                    <p className="text-sm text-gray-600">Show message content in notifications</p>
                                </div>
                                <Switch
                                    checked={settings.messagePreview}
                                    onCheckedChange={(checked) => updateSetting("messagePreview", checked)}
                                />
                            </div>
                        </div>
                    </div>

                    {/* Appearance */}
                    <div className="bg-white border border-gray-200 rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-black mb-4 flex items-center gap-2">
                            <Moon className="h-5 w-5" />
                            Appearance
                        </h2>
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Dark Mode</h3>
                                    <p className="text-sm text-gray-600">Switch to dark theme</p>
                                </div>
                                <Switch checked={settings.darkMode} onCheckedChange={(checked) => updateSetting("darkMode", checked)} />
                            </div>
                        </div>
                    </div>

                    {/* Privacy & Security */}
                    <div className="bg-white border border-gray-200 rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-black mb-4 flex items-center gap-2">
                            <Shield className="h-5 w-5" />
                            Privacy & Security
                        </h2>
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Read Receipts</h3>
                                    <p className="text-sm text-gray-600">Let others know when you've read their messages</p>
                                </div>
                                <Switch
                                    checked={settings.readReceipts}
                                    onCheckedChange={(checked) => updateSetting("readReceipts", checked)}
                                />
                            </div>
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Online Status</h3>
                                    <p className="text-sm text-gray-600">Show when you're online</p>
                                </div>
                                <Switch
                                    checked={settings.onlineStatus}
                                    onCheckedChange={(checked) => updateSetting("onlineStatus", checked)}
                                />
                            </div>
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Two-Factor Authentication</h3>
                                    <p className="text-sm text-gray-600">Add an extra layer of security</p>
                                </div>
                                <Switch
                                    checked={settings.twoFactorAuth}
                                    onCheckedChange={(checked) => updateSetting("twoFactorAuth", checked)}
                                />
                            </div>
                        </div>
                    </div>

                    {/* Data & Storage */}
                    <div className="bg-white border border-gray-200 rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-black mb-4 flex items-center gap-2">
                            <Download className="h-5 w-5" />
                            Data & Storage
                        </h2>
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Auto-download Media</h3>
                                    <p className="text-sm text-gray-600">Automatically download photos and videos</p>
                                </div>
                                <Switch
                                    checked={settings.autoDownload}
                                    onCheckedChange={(checked) => updateSetting("autoDownload", checked)}
                                />
                            </div>
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="font-medium text-black">Storage Usage</h3>
                                    <p className="text-sm text-gray-600">Manage your chat data and media</p>
                                </div>
                                <Button variant="ghost" size="sm" className="hover:bg-gray-100">
                                    View Details
                                </Button>
                            </div>
                        </div>
                    </div>

                    {/* Account Actions */}
                    <div className="bg-white border border-gray-200 rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-black mb-4">Account</h2>
                        <div className="space-y-3">
                            <Button variant="ghost" className="w-full justify-start h-12 px-4 hover:bg-gray-100 text-black">
                                <Download className="h-4 w-4 mr-3 text-gray-600" />
                                Export Chat Data
                            </Button>
                            <Button
                                variant="ghost"
                                className="w-full justify-start h-12 px-4 hover:bg-red-50 text-red-600 hover:text-red-700"
                            >
                                <Trash2 className="h-4 w-4 mr-3" />
                                Delete Account
                            </Button>
                            <Button
                                variant="ghost"
                                className="w-full justify-start h-12 px-4 hover:bg-gray-100 text-black"
                                onClick={logout}
                            >
                                <LogOut className="h-4 w-4 mr-3 text-gray-600" />
                                Sign Out
                            </Button>
                        </div>
                    </div>

                    {/* App Info */}
                    <div className="bg-white border border-gray-200 rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-black mb-4">About</h2>
                        <div className="space-y-3 text-sm">
                            <div className="flex justify-between">
                                <span className="text-gray-600">Version</span>
                                <span className="text-black">1.0.0</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-600">Last Updated</span>
                                <span className="text-black">Dec 15, 2024</span>
                            </div>
                            <div className="pt-3 border-t border-gray-200">
                                <Button variant="ghost" size="sm" className="hover:bg-gray-100 text-black">
                                    Terms of Service
                                </Button>
                                <Button variant="ghost" size="sm" className="hover:bg-gray-100 text-black ml-2">
                                    Privacy Policy
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}
