"use client"

import { useState, useEffect } from "react"
import { useAuth } from '@/hooks/useAuth';
import { useRouter } from 'next/navigation';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { ArrowLeft, Edit, Camera, Mail, Phone, MapPin, Calendar } from "lucide-react"
import Link from "next/link"

export default function ProfilePage() {
    const { isAuthenticated, loading, user, logout } = useAuth();
    const router = useRouter();
    const [isEditing, setIsEditing] = useState(false)
    const [profile, setProfile] = useState({
        name: user?.username || "John Doe",
        email: user?.email || "john.doe@example.com",
        phone: "+1 (555) 123-4567",
        location: "San Francisco, CA",
        bio: "Product designer passionate about creating intuitive user experiences.",
        joinDate: "January 2023",
    })

    useEffect(() => {
        if (!loading && !isAuthenticated) {
            router.push('/login');
        }
        
        if (user) {
            setProfile(prev => ({
                ...prev,
                name: user.username || prev.name,
                email: user.email || prev.email,
            }));
        }
    }, [isAuthenticated, loading, router, user]);

    if (loading) return <div className="h-screen bg-white flex items-center justify-center"><div className="text-black">Loading...</div></div>;
    if (!isAuthenticated) return null;

    const getInitials = (name: string) => name.split(' ').map(n => n[0]).join('').toUpperCase();

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
                        <h1 className="text-xl font-semibold text-black">Profile</h1>
                        <div className="ml-auto">
                            <Button onClick={() => setIsEditing(!isEditing)} variant="ghost" size="sm" className="hover:bg-gray-100">
                                <Edit className="h-4 w-4 mr-2 text-gray-600" />
                                {isEditing ? "Cancel" : "Edit"}
                            </Button>
                        </div>
                    </div>
                </div>
            </div>

            {/* Profile Content */}
            <div className="max-w-4xl mx-auto px-4 py-8">
                <div className="bg-white border border-gray-200 rounded-lg p-6">
                    {/* Avatar Section */}
                    <div className="flex flex-col items-center mb-8">
                        <div className="relative">
                            <Avatar className="h-24 w-24 border-2 border-gray-200">
                                <AvatarFallback className="bg-gray-100 text-black text-2xl font-medium">
                                    {getInitials(profile.name)}
                                </AvatarFallback>
                            </Avatar>
                            {isEditing && (
                                <Button
                                    size="sm"
                                    className="absolute -bottom-2 -right-2 h-8 w-8 p-0 bg-black hover:bg-gray-800 text-white rounded-full"
                                >
                                    <Camera className="h-4 w-4" />
                                </Button>
                            )}
                        </div>
                        <h2 className="text-2xl font-semibold text-black mt-4">{profile.name}</h2>
                        <p className="text-gray-600 mt-1">{profile.bio}</p>
                    </div>

                    {/* Profile Details */}
                    <div className="space-y-6">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div className="space-y-2">
                                <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                    <Mail className="h-4 w-4" />
                                    Email
                                </label>
                                {isEditing ? (
                                    <Input
                                        value={profile.email}
                                        onChange={(e) => setProfile({ ...profile, email: e.target.value })}
                                        className="bg-gray-50 border-gray-200 focus-visible:ring-black"
                                    />
                                ) : (
                                    <p className="text-black">{profile.email}</p>
                                )}
                            </div>

                            <div className="space-y-2">
                                <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                    <Phone className="h-4 w-4" />
                                    Phone
                                </label>
                                {isEditing ? (
                                    <Input
                                        value={profile.phone}
                                        onChange={(e) => setProfile({ ...profile, phone: e.target.value })}
                                        className="bg-gray-50 border-gray-200 focus-visible:ring-black"
                                    />
                                ) : (
                                    <p className="text-black">{profile.phone}</p>
                                )}
                            </div>

                            <div className="space-y-2">
                                <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                    <MapPin className="h-4 w-4" />
                                    Location
                                </label>
                                {isEditing ? (
                                    <Input
                                        value={profile.location}
                                        onChange={(e) => setProfile({ ...profile, location: e.target.value })}
                                        className="bg-gray-50 border-gray-200 focus-visible:ring-black"
                                    />
                                ) : (
                                    <p className="text-black">{profile.location}</p>
                                )}
                            </div>

                            <div className="space-y-2">
                                <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                    <Calendar className="h-4 w-4" />
                                    Joined
                                </label>
                                <p className="text-black">{profile.joinDate}</p>
                            </div>
                        </div>

                        {isEditing && (
                            <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
                                <Button variant="ghost" onClick={() => setIsEditing(false)} className="hover:bg-gray-100">
                                    Cancel
                                </Button>
                                <Button onClick={() => setIsEditing(false)} className="bg-black hover:bg-gray-800 text-white">
                                    Save Changes
                                </Button>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}
