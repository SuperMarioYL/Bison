import { useQuery } from '@tanstack/react-query';
import { getFeatures, Features } from '../services/api';

export function useFeatures() {
  return useQuery<Features>({
    queryKey: ['features'],
    queryFn: async () => {
      const { data } = await getFeatures();
      return data;
    },
  });
}

