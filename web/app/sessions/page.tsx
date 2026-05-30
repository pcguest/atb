import React from 'react';
import SessionList from '../../components/SessionList';
import ActorSessions from '../../components/ActorSessions';
import SchemaStatus from '../../components/SchemaStatus';

const SessionsPage: React.FC = () => {
  return (
    <div className="container mx-auto p-4">
      <h1 className="text-3xl font-bold text-gray-100 mb-6">Session Overview</h1>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <SessionList />
        <ActorSessions />
      </div>
      <div className="mt-6">
        <SchemaStatus />
      </div>
    </div>
  );
};

export default SessionsPage;
