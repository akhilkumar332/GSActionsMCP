import { useEffect } from 'react';

/**
 * useSSE - A hook to consume Server-Sent Events with auto-reconnect
 * @param {Function} onEvent - Callback function called with parsed event data
 */
export const useSSE = (onEvent) => {
  useEffect(() => {
    let eventSource;
    let reconnectTimeout;

    const connect = () => {
      // console.log('Connecting to SSE...');
      eventSource = new EventSource('/sse', { withCredentials: true });

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onEvent(data);
        } catch (err) {
          console.error('Failed to parse SSE event', err);
        }
      };

      eventSource.onerror = (err) => {
        console.error('SSE connection error:', err);
        eventSource.close();
        
        // Simple reconnect with 3 second delay
        reconnectTimeout = setTimeout(connect, 3000);
      };

      eventSource.onopen = () => {
        // SSE connection established
      };
    };

    connect();

    return () => {
      if (eventSource) {
        eventSource.close();
      }
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
    };
  }, [onEvent]);
};
