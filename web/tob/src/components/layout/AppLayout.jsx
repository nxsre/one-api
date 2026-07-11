import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import Topbar from './Topbar';

export default function AppLayout() {
  return (
    <div className="tob-app">
      <Sidebar />
      <div className="tob-main">
        <Topbar />
        <div className="tob-content page-enter">
          <Outlet />
        </div>
      </div>
    </div>
  );
}
