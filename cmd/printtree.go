package main
import (
  "fmt"
  cli "idealista-pp-cli/internal/cli"
)
func main(){
  root:=cli.RootCmd()
  for _, c := range root.Commands(){
    if c.Name()=="search"{
      fmt.Println("search short:", c.Short)
      fmt.Println("search long:", c.Long)
      for _, sc := range c.Commands(){
        fmt.Println("sub:", sc.Name(), "use:", sc.Use, "hidden:", sc.Hidden)
      }
    }
  }
}
