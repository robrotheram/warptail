import { ArrowLeft } from "lucide-react"
import { Button } from "../ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card"
import { useRouter } from "@tanstack/react-router";


export const ErrorCard = () => {
    const router = useRouter();
    return <Card className="col-span-2 my-10 mx-auto max-w-screen-sm">
        <CardHeader className="pb-2 flex flex-row items-center justify-center gap-4">
            <CardTitle className="text-2xl">Unable to load service</CardTitle>
        </CardHeader>
        <CardContent className='mt-4 px-16'>
            <Button className="w-full" onClick={() => router.history.back()}>
                <ArrowLeft className="mr-2" /> Back
            </Button>
        </CardContent>
    </Card>
}