// Shared type definitions for the chat application

export interface User {
  id: number;
  username: string;
  email: string;
  role: string;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: number;
  content: string;
  user_id: number;
  user: User;
  room_id: number;
  created_at: string;
  updated_at: string;
}

export interface Room {
  id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  role?: string;
}