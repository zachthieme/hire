import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notifications as notifApi, type Notification } from '@/lib/api'
import { Bell } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useNavigate } from 'react-router-dom'

export default function NotificationBell() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: notifs = [] } = useQuery({ queryKey: ['notifications'], queryFn: () => notifApi.list(), refetchInterval: 15000 })
  const markRead = useMutation({
    mutationFn: notifApi.markRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })

  const unreadCount = notifs.filter((n: Notification) => !n.read).length

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="relative p-2">
        <Bell className="h-5 w-5" />
        {unreadCount > 0 && (
          <Badge variant="destructive" className="absolute -top-1 -right-1 h-5 w-5 flex items-center justify-center p-0 text-xs">
            {unreadCount}
          </Badge>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        {notifs.length === 0 && (
          <DropdownMenuItem disabled>No notifications</DropdownMenuItem>
        )}
        {notifs.map((n: Notification) => (
          <DropdownMenuItem
            key={n.id}
            className={n.read ? 'opacity-60' : 'font-medium'}
            onClick={() => {
              if (!n.read) markRead.mutate(n.id)
              if (n.link) navigate(n.link)
            }}
          >
            {n.message}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
