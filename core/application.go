package core

import (
	"reflect"
	"strings"
	"sync"
)

func NewApp(obj interface{}) (*Application, error) {
	value := correction(reflect.ValueOf(obj))
	if value.Kind() != reflect.Struct {
		return nil, ErrorStructType
	}
	app := &Application{Origin: obj, Parse: make(map[string]interface{}), StructName: value.Type().Name()}
	res := parseStruct(obj)
	app.Parse = res
	return app, nil
}
func (a *Application) Overflow(arr []string) {
	for _, item := range arr {
		if strings.Contains(item, ".") {
			arr := strings.Split(item, ".")
			idx := int64(0)
			roll(arr, a.Parse, &idx)
		} else {
			delete(a.Parse, item)
		}
	}
}
func (a *Application) Result() map[string]interface{} {
	return a.Parse
}

func parseStruct(obj interface{}) map[string]interface{} {
	value := correction(reflect.ValueOf(obj))
	types := value.Type()
	// 解析到Parse 字段
	result := make(map[string]interface{})
	ch := make(chan map[string]interface{})
	defer close(ch)
	go worker(ch, result)
	wg := sync.WaitGroup{}
	wg.Add(value.NumField())
	for i := 0; i < value.NumField(); i++ {
		go func(v reflect.Value, i int) {
			//v := value.Field(i)
			tag := types.Field(i).Tag.Get(TAGNAME)
			if strings.Contains(tag, ",") {
				tag = strings.TrimSpace(strings.Split(tag, ",")[0])
			}
			switch v.Kind() {
			case reflect.Ptr:
				res := parseStruct(v.Interface())
				//result[tag] = res
				ch <- map[string]interface{}{tag: res}
			case reflect.Slice:
				if !v.IsNil() && v.Index(0).Kind() != reflect.Struct && v.Index(0).Kind() != reflect.Ptr {
					//result[tag] = v.Interface()
					ch <- map[string]interface{}{tag: v.Interface()}
				} else {
					wg := sync.WaitGroup{}
					wg.Add(v.Len())
					slice := make([]map[string]interface{}, 0)
					for i := 0; i < v.Len(); i++ {
						go func(item reflect.Value) {
							res := parseStruct(item.Interface())
							slice = append(slice, res)
							wg.Done()
						}(v.Index(i))
					}
					wg.Wait()
					//result[tag] = slice
					ch <- map[string]interface{}{tag: slice}
				}
			default:
				//result[tag] = v.Interface()
				ch <- map[string]interface{}{tag: v.Interface()}
			}

			wg.Done()
		}(value.Field(i), i)
	}
	wg.Wait()
	ch <- nil
	return result
}
func roll(arr []string, sc map[string]interface{}, idx *int64) {
	if *idx < int64(len(arr)-1) {
		d := sc
		m := d[arr[*idx]].(map[string]interface{})
		next := *idx + 1
		roll(arr[*idx:], m, &next)
	} else {
		delete(sc, arr[*idx])
	}
}

func correction(value reflect.Value) reflect.Value {
	for value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	return value
}

func worker(ch chan map[string]interface{}, result map[string]interface{}) {
	for {
		select {
		case res, ok := <-ch:
			if !ok {
				//wg.Done()
				return
			}
			for k, v := range res {
				result[k] = v
			}
		}
	}
}
