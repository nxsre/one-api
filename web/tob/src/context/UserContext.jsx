import React, { createContext, useContext, useMemo, useState } from 'react';
import { getStoredUser } from '@/lib/auth';

const UserContext = createContext(null);

export function UserProvider({ children }) {
  const [user, setUser] = useState(() => getStoredUser());

  const value = useMemo(
    () => ({
      user,
      setUser,
      login: (payload) => {
        localStorage.setItem('user', JSON.stringify(payload));
        setUser(payload);
      },
      logout: () => {
        localStorage.removeItem('user');
        setUser(null);
      },
    }),
    [user]
  );

  return <UserContext.Provider value={value}>{children}</UserContext.Provider>;
}

export function useUser() {
  const ctx = useContext(UserContext);
  if (!ctx) throw new Error('useUser must be used within UserProvider');
  return ctx;
}
