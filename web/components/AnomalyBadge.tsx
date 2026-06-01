import React from 'react';

interface AnomalyBadgeProps {
  flag: string;
}

const AnomalyBadge: React.FC<AnomalyBadgeProps> = ({ flag }) => {
  let icon: React.ReactNode;
  let tooltipText: string;
  let className: string = "inline-block w-4 h-4 mr-1"; // Basic styling

  switch (flag) {
    case 'tool_without_approval':
      icon = (
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="text-amber-500">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
        </svg>
      );
      tooltipText = 'Tool call without preceding human approval.';
      break;
    case 'unresolved_identity':
      icon = (
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="text-gray-500">
          <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z" />
        </svg>
      );
      tooltipText = 'Actor identity is unresolved (e.g., API key).';
      break;
    case 'policy_denied_executed':
      icon = (
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="text-red-500">
          <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 0 0 5.636 5.636m12.728 12.728A9 9 0 0 1 5.636 5.636m12.728 12.728L5.636 5.636" />
        </svg>
      );
      tooltipText = 'Policy denied the action, but it executed anyway.';
      break;
    case 'action_failed':
      icon = (
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="text-orange-500">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
        </svg>
      );
      tooltipText = 'A privileged action did not succeed (ai.action.error).';
      break;
    case 'session_not_closed':
      icon = (
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="text-red-500">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
      );
      tooltipText = 'Session has not been explicitly closed.';
      break;
    default:
      icon = null;
      tooltipText = `Unknown anomaly: ${flag}`;
  }

  if (!icon) {
    return null;
  }

  return (
    <span className="relative group" title={tooltipText}>
      {icon}
      {/* Optional: A more sophisticated tooltip could be implemented here */}
    </span>
  );
};

export default AnomalyBadge;
