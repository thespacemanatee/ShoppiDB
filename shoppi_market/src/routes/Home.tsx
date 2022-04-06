import { Breadcrumb } from "antd"

export default function VideoSurveillance() {
  return (
    <div className="h-full min-h-screen">
      <Breadcrumb>
        <Breadcrumb.Item>Marketplace</Breadcrumb.Item>
      </Breadcrumb>
      <div className="mt-4 grid grid-cols-3 gap-4">
        {Array(5)
          .fill(0)
          .map((e, idx) => (
            <div
              key={idx}
              className="flex flex-col items-center justify-center"
            />
          ))}
      </div>
    </div>
  )
}
