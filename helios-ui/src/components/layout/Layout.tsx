import { Outlet, NavLink } from 'react-router-dom';
import { 
  LayoutDashboard, 
  Box, 
  Image, 
  HardDrive, 
  Network,
  Activity 
} from 'lucide-react';
import { cn } from '../../lib/utils';

const navigation = [
  { name: 'Dashboard', to: '/dashboard', icon: LayoutDashboard },
  { name: 'Containers', to: '/containers', icon: Box },
  { name: 'Images', to: '/images', icon: Image },
  { name: 'Volumes', to: '/volumes', icon: HardDrive },
  { name: 'Networks', to: '/networks', icon: Network },
];

export default function Layout() {
  return (
    <div className="flex h-screen bg-gray-900">
      {/* Sidebar */}
      <aside className="w-64 bg-gray-800 border-r border-gray-700">
        <div className="flex items-center gap-2 p-6 border-b border-gray-700">
          <Activity className="w-8 h-8 text-blue-500" />
          <span className="text-xl font-bold text-white">Helios</span>
        </div>
        <nav className="p-4 space-y-1">
          {navigation.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 px-4 py-3 rounded-lg transition-colors',
                  isActive
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                )
              }
            >
              <item.icon className="w-5 h-5" />
              <span className="font-medium">{item.name}</span>
            </NavLink>
          ))}
        </nav>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto p-8">
        <Outlet />
      </main>
    </div>
  );
}
