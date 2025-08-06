"use client";
import { useState, useEffect } from 'react';
import api from '@/lib/api';
import { User } from '@/lib/types';

export const useAuth = () => {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (token) {
      // Fetch user profile to get user info including role
      fetchUserProfile();
    } else {
      setLoading(false);
    }
  }, []);

  const fetchUserProfile = async () => {
    try {
      const response = await api.get('/auth/profile');
      setUser(response.data);
      setIsAuthenticated(true);
    } catch (error) {
      // Token might be invalid
      localStorage.removeItem('token');
      setIsAuthenticated(false);
    } finally {
      setLoading(false);
    }
  };

  const login = (token: string, userData?: User) => {
    localStorage.setItem('token', token);
    setIsAuthenticated(true);
    if (userData) {
      setUser(userData);
    } else {
      // Fetch user profile if not provided
      fetchUserProfile();
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setIsAuthenticated(false);
    setUser(null);
  };

  return {
    isAuthenticated,
    loading,
    user,
    login,
    logout
  };
}; 