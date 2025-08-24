"use client"

import { useEffect, use } from "react"
import { useAuth } from '@/hooks/useAuth';
import { useRouter } from 'next/navigation';
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { ArrowLeft, MessageCircle, Phone, Video, Mail, MapPin, Calendar } from "lucide-react"
import Link from "next/link"

// Mock user data - in real app this would come from API
const getUserData = (id: string) => {
    const users: Record<string, any> = {
        "1": {
            name: "Sarah Wilson",
            avatar: "SW",
            email: "sarah.wilson@example.com",
            phone: "+1 (555) 987-6543",
            location: "New York, NY",
            bio: "Senior UX Designer with 8+ years of experience in creating user-centered digital experiences.",
            joinDate: "March 2022",
            online: true,
            lastSeen: "Active now",
        },
        "2": {
            name: "Design Team",
            avatar: "DT",
            email: "design@company.com",
            location: "Remote Team",
            bio: "Creative design team focused on innovative solutions and beautiful interfaces.",
            joinDate: "January 2021",
            online: false,
            lastSeen: "2 hours ago",
        },
        "3": {
            name: "Alex Chen",
            avatar: "AC",
            email: "alex.chen@example.com",
            phone: "+1 (555) 456-7890",
            location: "San Francisco, CA",
            bio: "Full-stack developer passionate about building scalable web applications.",
            joinDate: "June 2023",
            online: true,
            lastSeen: "Active now",
        },
        "4": {
            name: "Marketing",
            avatar: "MK",
            email: "marketing@company.com",
            location: "Los Angeles, CA",
            bio: "Marketing team driving growth and brand awareness.",
            joinDate: "February 2021",
            online: false,
            lastSeen: "1 day ago",
        },
        "5": {
            name: "David Kim",
            avatar: "DK",
            email: "david.kim@example.com",
            phone: "+1 (555) 234-5678",
            location: "Seattle, WA",
            bio: "Product manager with a focus on user experience and data-driven decisions.",
            joinDate: "September 2022",
            online: false,
            lastSeen: "3 hours ago",
        },
        "6": {
            name: "Product Team",
            avatar: "PT",
            email: "product@company.com",
            location: "Austin, TX",
            bio: "Product team building the future of digital experiences.",
            joinDate: "May 2020",
            online: false,
            lastSeen: "Yesterday",
        },
    }
    return users[id] || users["1"]
}

export default function UserProfilePage({ params }: { params: Promise<{ id: string }> }) {
    const { isAuthenticated, loading, user } = useAuth();
    const router = useRouter();
    const { id } = use(params);
    const userData = getUserData(id)

    useEffect(() => {
        if (!loading && !isAuthenticated) {
            router.push('/login');
        }
    }, [isAuthenticated, loading, router]);

    if (loading) return <div className="h-screen bg-white flex items-center justify-center"><div className="text-black">Loading...</div></div>;
    if (!isAuthenticated) return null;

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
                        <h1 className="text-xl font-semibold text-black">{userData.name}</h1>
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
                                <AvatarFallback className="bg-gray-100 text-black text-2xl font-medium">{userData.avatar}</AvatarFallback>
                            </Avatar>
                            {userData.online && (
                                <div className="absolute -bottom-1 -right-1 h-6 w-6 bg-black border-3 border-white rounded-full" />
                            )}
                        </div>
                        <h2 className="text-2xl font-semibold text-black mt-4">{userData.name}</h2>
                        <p className="text-gray-600 mt-1">{userData.bio}</p>
                        <p className="text-sm text-gray-500 mt-2">{userData.online ? "Active now" : `Last seen ${userData.lastSeen}`}</p>
                    </div>

                    {/* Action Buttons */}
                    <div className="flex justify-center gap-3 mb-8">
                        <Link href="/chat">
                            <Button className="bg-black hover:bg-gray-800 text-white">
                                <MessageCircle className="h-4 w-4 mr-2" />
                                Message
                            </Button>
                        </Link>
                        <Button variant="outline" className="border-gray-200 hover:bg-gray-50 bg-transparent">
                            <Phone className="h-4 w-4 mr-2" />
                            Call
                        </Button>
                        <Button variant="outline" className="border-gray-200 hover:bg-gray-50 bg-transparent">
                            <Video className="h-4 w-4 mr-2" />
                            Video
                        </Button>
                    </div>

                    {/* Profile Details */}
                    <div className="space-y-6">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                            {userData.email && (
                                <div className="space-y-2">
                                    <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                        <Mail className="h-4 w-4" />
                                        Email
                                    </label>
                                    <p className="text-black">{userData.email}</p>
                                </div>
                            )}

                            {userData.phone && (
                                <div className="space-y-2">
                                    <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                        <Phone className="h-4 w-4" />
                                        Phone
                                    </label>
                                    <p className="text-black">{userData.phone}</p>
                                </div>
                            )}

                            {userData.location && (
                                <div className="space-y-2">
                                    <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                        <MapPin className="h-4 w-4" />
                                        Location
                                    </label>
                                    <p className="text-black">{userData.location}</p>
                                </div>
                            )}

                            <div className="space-y-2">
                                <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                                    <Calendar className="h-4 w-4" />
                                    Joined
                                </label>
                                <p className="text-black">{userData.joinDate}</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}
