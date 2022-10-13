import { Route, Routes, Navigate } from 'react-router-dom';

import CreateSecret from './createSecret/CreateSecret';
import DisplaySecret from './displaySecret/DisplaySecret';
import Upload from './createSecret/Upload';
import Login from './login/Login';
import { getUser } from './utils/utils';

export const Routing = () => {
  const user = getUser()
  console.log(user)
  return (
    <Routes>
      <Route path="/" element={
        <AdminRoute user={user}>
          <CreateSecret />
        </AdminRoute>
      } />
      <Route path="/login" element={<Login/>} />
      <Route path="/upload" element={
        <AdminRoute user={user}>
          <Upload />
        </AdminRoute>
      } />
      <Route path="/:format/:key/:password" element={<DisplaySecret />} />
      <Route path="/:format/:key" element={<DisplaySecret />} />
    </Routes>
  );
};

export const AdminRoute = (props: any) => {
  console.log(props)
  if (!props.user) {
    return <Navigate replace to="/login" />
  }
  return props.children;
};