
import * as React from "react"
import { Check, ChevronsUpDown } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useQuery } from "@tanstack/react-query"
import { getTailScaleNodes, Route } from "@/lib/api"
import { Label } from "../ui/label"
import { Input } from "../ui/input"

type TailscaleHostProps = {
  route: Route
  updateRoute: (route: Route) => void
}

export const TailscaleHost = ({ route, updateRoute }: TailscaleHostProps) => {
  const { machine } = route
  const [open, setOpen] = React.useState(false)
  const [inputValue, setInputValue] = React.useState("")
  const { data } = useQuery({
    queryKey: ['tailscaleNodes'],
    queryFn: getTailScaleNodes,
  })
  const options = React.useMemo(() => {
    if (data) {
      let options = data.map((node) => ({
        label: node.hostname,
        value: node.ip,
      }))
      if (machine?.address) {
        return options.find((option) => option.value === machine.address) ?
          options :
          [...options, machine.node ? { label: machine.node, value: machine.address } : { label: machine.address, value: machine.address }]
      }
      return options
    }
    return []
  }, [data, machine])


  const handleInputChange = (input: string) => {
    setInputValue(input)
  }


  const handlePortChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = e.target
    const port = value === '' ? 0 : parseInt(value, 10)
    if (!isNaN(port)) {
      updateRoute({
        ...route, machine: {
          ...route.machine,
          port: port,
        }
      })
    }
  }

  const handleNodeChange = (value: string) => {
    console.log("Node Change", value)
    const _machine = options.find((option) => option.value === value)
    updateRoute({
      ...route, machine: {
        ...route.machine,
        node: _machine?.label,
        address: _machine?.value ?? value,
      }
    })
  }

  const addCustomValue = () => {
    if (inputValue && !options.some((option) => option.value === inputValue.toLowerCase())) {
      setInputValue(inputValue.toLowerCase())
      updateRoute({
        ...route, machine: {
          ...route.machine,
          address: inputValue.toLowerCase()
        }
      })
    }
    setOpen(false)
  }


  return (
    <div className="md:col-span-5 grid md:grid-cols-2 col-span-6 gap-4 w-full">
      <div>
        <Label htmlFor="host">Tailscale Host</Label>
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button variant="outline" role="combobox" aria-expanded={open} className="w-full justify-between">
              {machine?.address ? options.find((option) => option.value === machine.address)?.label || inputValue : "Select node..."}
              <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-full p-0">
            <Command>
              <CommandInput placeholder="Search node..." value={inputValue} onValueChange={handleInputChange} />
              <CommandList>
                <CommandEmpty>
                  <div className="px-2 text-sm">
                    <Button
                      // variant="ghost"
                      className="mt-2 w-full justify-start text-sm font-normal"
                      onClick={addCustomValue}
                    >
                      Add "{inputValue}"
                    </Button>
                  </div>
                </CommandEmpty>
              </CommandList>
              <CommandList>
                <CommandGroup>
                  {options.map((option) => (
                    <CommandItem
                      key={option.value}
                      value={option.value}
                      onSelect={(currentValue) => {
                        handleNodeChange(currentValue)
                        setOpen(false)
                      }}
                    >
                      <Check className={cn("mr-2 h-4 w-4", machine?.address === option.value ? "opacity-100" : "opacity-0")} />
                      {option.label}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      </div>
      <div>
        <Label htmlFor="port">Tailscale Port</Label>
        <Input
          id="port"
          name="port"
          type="text"
          value={machine?.port}
          onChange={handlePortChange}
        />
      </div>
    </div>
  )
}